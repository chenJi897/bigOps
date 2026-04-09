// Package scriptguard 提供脚本危险命令检测，在执行前拦截高危操作。
package scriptguard

import (
	"fmt"
	"regexp"
	"strings"
)

// rule 一条检测规则。
type rule struct {
	name    string
	pattern *regexp.Regexp
}

// bashRules 针对 bash/shell 脚本的危险命令正则。
var bashRules = []rule{
	{"rm -rf /", regexp.MustCompile(`rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)*/([\s;|&]|$)`)},
	{"rm critical dirs", regexp.MustCompile(`rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)*/(?:etc|var|usr|boot|sys|proc|home|root|lib|lib64|sbin|bin)([\s/;|&]|$)`)},
	{"mkfs", regexp.MustCompile(`\bmkfs\b`)},
	{"dd to device", regexp.MustCompile(`\bdd\b.*\bof\s*=\s*/dev/`)},
	{"fork bomb", regexp.MustCompile(`:\(\)\s*\{.*\|.*&.*\}\s*;?\s*:`)},
	{"shutdown/reboot", regexp.MustCompile(`\b(?:shutdown|reboot|poweroff|halt)\b`)},
	{"init 0/6", regexp.MustCompile(`\binit\s+[06]\b`)},
	{"chmod 777 /", regexp.MustCompile(`\bchmod\s+(-[a-zA-Z]+\s+)*777\s+/`)},
	{"chown root /", regexp.MustCompile(`\bchown\s+(-R\s+)?\S+\s+/\s*$`)},
	{"overwrite disk", regexp.MustCompile(`>\s*/dev/sd[a-z]`)},
	{"pipe to shell", regexp.MustCompile(`(?:curl|wget)\b.*\|\s*(?:ba)?sh`)},
}

// Validate 检测脚本内容是否包含危险命令。
// scriptType: bash / python / powershell
// 返回 nil 表示安全，返回 error 表示命中了危险规则。
func Validate(scriptContent, scriptType string) error {
	lower := strings.ToLower(strings.TrimSpace(scriptContent))
	if lower == "" {
		return nil
	}

	switch scriptType {
	case "python":
		return validatePython(lower)
	case "powershell":
		// powershell 暂不检测
		return nil
	default:
		return validateBash(lower)
	}
}

func validateBash(content string) error {
	for _, r := range bashRules {
		if r.pattern.MatchString(content) {
			return fmt.Errorf("危险命令检测：匹配规则 [%s]，脚本被拦截", r.name)
		}
	}
	return nil
}

// validatePython 检测 python 脚本中通过 os.system / subprocess 调用的危险命令。
func validatePython(content string) error {
	// 提取 os.system("...") 和 subprocess.run/call/Popen(["..."]) 中的字符串参数
	pyShellPattern := regexp.MustCompile(`(?:os\.system|subprocess\.(?:run|call|Popen|check_output|check_call))\s*\(\s*["\x60'](.*?)["\x60']`)
	matches := pyShellPattern.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) > 1 {
			if err := validateBash(m[1]); err != nil {
				return err
			}
		}
	}
	return nil
}
