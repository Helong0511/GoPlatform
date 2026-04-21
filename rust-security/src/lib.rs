use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes128Gcm, Aes256Gcm, Nonce};
use base64::Engine as _;
use jsonwebtoken::{decode, encode, Algorithm, DecodingKey, EncodingKey, Header, Validation};
use rand::RngCore;
use serde::{Deserialize, Serialize};
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::time::{SystemTime, UNIX_EPOCH};
use std::{panic, ptr};

#[derive(Debug, Serialize, Deserialize)]
struct Claims {
    sub: String,
    role: String,
    exp: usize,
}

fn c_ptr_to_string(ptr: *const c_char) -> Option<String> {
    if ptr.is_null() {
        return None;
    }
    unsafe { CStr::from_ptr(ptr).to_str().ok().map(|s| s.to_owned()) }
}

fn to_c_string(value: String) -> *mut c_char {
    match CString::new(value) {
        Ok(s) => s.into_raw(),
        Err(_) => ptr::null_mut(),
    }
}

fn decode_key_bytes(key_raw: &str) -> Option<Vec<u8>> {
    if key_raw.is_empty() {
        return None;
    }

    if let Ok(decoded) = base64::engine::general_purpose::STANDARD.decode(key_raw) {
        if !decoded.is_empty() {
            return Some(decoded);
        }
    }

    Some(key_raw.as_bytes().to_vec())
}

fn encrypt_with_aes_gcm(data: &[u8], key: &[u8]) -> Option<String> {
    let mut nonce_bytes = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);

    let ciphertext = match key.len() {
        16 => {
            let cipher = Aes128Gcm::new_from_slice(key).ok()?;
            cipher.encrypt(nonce, data).ok()?
        }
        32 => {
            let cipher = Aes256Gcm::new_from_slice(key).ok()?;
            cipher.encrypt(nonce, data).ok()?
        }
        _ => return None,
    };

    let mut output = Vec::with_capacity(nonce_bytes.len() + ciphertext.len());
    output.extend_from_slice(&nonce_bytes);
    output.extend_from_slice(&ciphertext);
    Some(hex::encode(output))
}

fn decrypt_with_aes_gcm(ciphertext_hex: &str, key: &[u8]) -> Option<String> {
    let raw = hex::decode(ciphertext_hex).ok()?;
    if raw.len() < 12 + 16 {
        return None;
    }

    let (nonce_raw, cipher_raw) = raw.split_at(12);
    let nonce = Nonce::from_slice(nonce_raw);

    let plaintext = match key.len() {
        16 => {
            let cipher = Aes128Gcm::new_from_slice(key).ok()?;
            cipher.decrypt(nonce, cipher_raw).ok()?
        }
        32 => {
            let cipher = Aes256Gcm::new_from_slice(key).ok()?;
            cipher.decrypt(nonce, cipher_raw).ok()?
        }
        _ => return None,
    };

    String::from_utf8(plaintext).ok()
}

#[no_mangle]
pub extern "C" fn generate_jwt(user_id_ptr: *const c_char, role_ptr: *const c_char, secret_ptr: *const c_char, expires_in_seconds: i64) -> *mut c_char {
    let result = panic::catch_unwind(|| {
        let user_id = c_ptr_to_string(user_id_ptr)?;
        let role = c_ptr_to_string(role_ptr)?;
        let secret = c_ptr_to_string(secret_ptr)?;

        let now = SystemTime::now().duration_since(UNIX_EPOCH).ok()?.as_secs() as i64;
        let exp = if expires_in_seconds > 0 { expires_in_seconds } else { 3600 };

        let claims = Claims {
            sub: user_id,
            role,
            exp: (now + exp) as usize,
        };

        let token = encode(&Header::new(Algorithm::HS256), &claims, &EncodingKey::from_secret(secret.as_bytes())).ok()?;
        Some(to_c_string(token))
    });

    match result {
        Ok(Some(ptr)) => ptr,
        _ => ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn validate_jwt(token_ptr: *const c_char, secret_ptr: *const c_char) -> bool {
    let result = panic::catch_unwind(|| {
        let token = c_ptr_to_string(token_ptr)?;
        let secret = c_ptr_to_string(secret_ptr)?;

        let validation = Validation::new(Algorithm::HS256);
        decode::<Claims>(&token, &DecodingKey::from_secret(secret.as_bytes()), &validation).ok()?;
        Some(true)
    });

    matches!(result, Ok(Some(true)))
}

#[no_mangle]
pub extern "C" fn encrypt(data: *const c_char, key: *const c_char) -> *mut c_char {
    let result = panic::catch_unwind(|| {
        let data_str = c_ptr_to_string(data)?;
        let key_str = c_ptr_to_string(key)?;
        let key_bytes = decode_key_bytes(&key_str)?;
        let encrypted = encrypt_with_aes_gcm(data_str.as_bytes(), &key_bytes)?;
        Some(to_c_string(encrypted))
    });

    match result {
        Ok(Some(ptr)) => ptr,
        _ => ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn decrypt(ciphertext_ptr: *const c_char, key_ptr: *const c_char) -> *mut c_char {
    let result = panic::catch_unwind(|| {
        let ciphertext = c_ptr_to_string(ciphertext_ptr)?;
        let key_raw = c_ptr_to_string(key_ptr)?;
        let key_bytes = decode_key_bytes(&key_raw)?;
        let plaintext = decrypt_with_aes_gcm(&ciphertext, &key_bytes)?;
        Some(to_c_string(plaintext))
    });

    match result {
        Ok(Some(ptr)) => ptr,
        _ => ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn free_string(s: *mut c_char) {
    if !s.is_null() {
        unsafe { let _ = CString::from_raw(s); };
    }
}
