package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadDotEnvFiles 按顺序尝试加载 .env；已存在的环境变量不会被覆盖。
// 返回成功加载的文件路径（首个），均不存在时返回空字符串。
func LoadDotEnvFiles(extraPaths ...string) (string, error) {
	paths := append(append([]string{}, extraPaths...), ".env", "config/.env")
	var loaded string
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if err := loadDotEnvFile(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return loaded, fmt.Errorf("load dotenv %s: %w", path, err)
		}
		if loaded == "" {
			loaded = path
		}
	}
	return loaded, nil
}

func loadDotEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("setenv %s: %w", key, err)
		}
	}
	return scanner.Err()
}
