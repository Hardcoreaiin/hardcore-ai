package toolchain

import (
	"fmt"
	"runtime"
)

type Tool string

const (
	ToolGCC  Tool = "arm-none-eabi-gcc"
	ToolQEMU Tool = "qemu-arm"
)

type release struct {
	baseURL  string
	version  string
	filename string // no extension
	ext      string // .tar.gz or .zip
}

func resolveRelease(t Tool) (release, error) {
	os, arch, err := xpackPlatform()
	if err != nil {
		return release{}, err
	}

	switch t {
	case ToolGCC:
		ver := "15.2.1-1.1"
		fname := fmt.Sprintf("xpack-arm-none-eabi-gcc-%s-%s-%s", ver, os, arch)
		ext := archiveExt()
		return release{
			baseURL:  "https://github.com/xpack-dev-tools/arm-none-eabi-gcc-xpack/releases/download/v" + ver,
			version:  ver,
			filename: fname,
			ext:      ext,
		}, nil

	case ToolQEMU:
		ver := "9.2.4-1"
		fname := fmt.Sprintf("xpack-qemu-arm-%s-%s-%s", ver, os, arch)
		ext := archiveExt()
		return release{
			baseURL:  "https://github.com/xpack-dev-tools/qemu-arm-xpack/releases/download/v" + ver,
			version:  ver,
			filename: fname,
			ext:      ext,
		}, nil

	default:
		return release{}, fmt.Errorf("unknown tool: %s", t)
	}
}

// xpackPlatform maps runtime.GOOS/GOARCH to xpack's OS and arch tokens.
func xpackPlatform() (os, arch string, err error) {
	switch runtime.GOOS {
	case "linux":
		os = "linux"
	case "darwin":
		os = "darwin"
	case "windows":
		os = "win32"
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "x64"
	case "arm64":
		if runtime.GOOS == "windows" {
			// xpack only ships win32-x64; ARM64 Windows can run it via emulation
			arch = "x64"
		} else {
			arch = "arm64"
		}
	default:
		return "", "", fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
	}

	return os, arch, nil
}

func archiveExt() string {
	if runtime.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
