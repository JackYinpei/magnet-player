package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ValidationError 验证错误类型
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ValidationErrors 多个验证错误
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var messages []string
	for _, err := range v {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// MagnetValidator 磁力链接验证器
type MagnetValidator struct{}

// ValidateMagnetURI 验证磁力链接
func (mv *MagnetValidator) ValidateMagnetURI(magnetURI string) error {
	if magnetURI == "" {
		return ValidationError{Field: "magnetUri", Message: "磁力链接不能为空"}
	}

	// 去除首尾空格
	magnetURI = strings.TrimSpace(magnetURI)

	// 检查是否以magnet:?开头
	if !strings.HasPrefix(magnetURI, "magnet:?") {
		return ValidationError{Field: "magnetUri", Message: "磁力链接必须以'magnet:?'开头"}
	}

	// 解析URL
	parsedURL, err := url.Parse(magnetURI)
	if err != nil {
		return ValidationError{Field: "magnetUri", Message: "磁力链接格式无效"}
	}

	// 检查查询参数
	queryParams := parsedURL.Query()
	
	// 必须包含xt参数（eXact Topic）
	xtParams := queryParams["xt"]
	if len(xtParams) == 0 {
		return ValidationError{Field: "magnetUri", Message: "磁力链接必须包含xt参数"}
	}

	// 检查xt参数是否为btih格式
	foundValidXt := false
	for _, xt := range xtParams {
		if strings.HasPrefix(xt, "urn:btih:") {
			// 提取hash值
			hash := strings.TrimPrefix(xt, "urn:btih:")
			if err := mv.validateInfoHash(hash); err != nil {
				return ValidationError{Field: "magnetUri", Message: fmt.Sprintf("无效的InfoHash: %v", err)}
			}
			foundValidXt = true
			break
		}
	}

	if !foundValidXt {
		return ValidationError{Field: "magnetUri", Message: "磁力链接必须包含有效的btih格式的xt参数"}
	}

	return nil
}

// validateInfoHash 验证InfoHash格式
func (mv *MagnetValidator) validateInfoHash(hash string) error {
	// InfoHash可以是40字符的十六进制字符串（SHA1）或32字符的base32编码
	if len(hash) == 40 {
		// 验证十六进制格式
		matched, _ := regexp.MatchString("^[a-fA-F0-9]{40}$", hash)
		if !matched {
			return fmt.Errorf("InfoHash必须是40字符的十六进制字符串")
		}
	} else if len(hash) == 32 {
		// 验证base32格式（通常为大写字母A-Z和数字2-7）
		matched, _ := regexp.MatchString("^[A-Z2-7]{32}$", hash)
		if !matched {
			return fmt.Errorf("InfoHash必须是32字符的base32字符串")
		}
	} else {
		return fmt.Errorf("InfoHash长度无效，应为40字符（十六进制）或32字符（base32）")
	}

	return nil
}

// FilePathValidator 文件路径验证器
type FilePathValidator struct{}

// ValidateFilePath 验证文件路径安全性
func (fpv *FilePathValidator) ValidateFilePath(filePath string) error {
	if filePath == "" {
		return ValidationError{Field: "filePath", Message: "文件路径不能为空"}
	}

	// 检查路径遍历攻击
	if strings.Contains(filePath, "..") {
		return ValidationError{Field: "filePath", Message: "文件路径不能包含'..'"}
	}

	// 检查绝对路径（在某些情况下可能不安全）
	if strings.HasPrefix(filePath, "/") || strings.Contains(filePath, ":") {
		return ValidationError{Field: "filePath", Message: "不允许绝对路径"}
	}

	// 检查危险字符
	dangerousChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		if strings.Contains(filePath, char) {
			return ValidationError{Field: "filePath", Message: fmt.Sprintf("文件路径不能包含字符'%s'", char)}
		}
	}

	return nil
}

// InfoHashValidator InfoHash验证器
type InfoHashValidator struct{}

// ValidateInfoHash 验证InfoHash
func (ihv *InfoHashValidator) ValidateInfoHash(infoHash string) error {
	if infoHash == "" {
		return ValidationError{Field: "infoHash", Message: "InfoHash不能为空"}
	}

	mv := &MagnetValidator{}
	return mv.validateInfoHash(infoHash)
}

// StringValidator 字符串验证器
type StringValidator struct{}

// ValidateRequired 验证必填字段
func (sv *StringValidator) ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{Field: fieldName, Message: fmt.Sprintf("%s不能为空", fieldName)}
	}
	return nil
}

// ValidateMaxLength 验证最大长度
func (sv *StringValidator) ValidateMaxLength(value, fieldName string, maxLength int) error {
	if len(value) > maxLength {
		return ValidationError{
			Field:   fieldName, 
			Message: fmt.Sprintf("%s长度不能超过%d个字符", fieldName, maxLength),
		}
	}
	return nil
}

// ValidateMinLength 验证最小长度
func (sv *StringValidator) ValidateMinLength(value, fieldName string, minLength int) error {
	if len(value) < minLength {
		return ValidationError{
			Field:   fieldName, 
			Message: fmt.Sprintf("%s长度不能少于%d个字符", fieldName, minLength),
		}
	}
	return nil
}