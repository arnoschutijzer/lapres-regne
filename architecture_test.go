package main

import (
	"os/exec"
	"strings"
	"testing"
)

const (
	module   = "lapresregne"
	adapters = module + "/internal/adapters/"
)

var domains = []string{module + "/internal/world", module + "/internal/soundtrack"}

// Domain code stays independent of the operating system and the network.
var infrastructure = []string{"os", "net", "syscall", "database"}

func isDomain(pkg string) bool {
	for _, domain := range domains {
		if pkg == domain || strings.HasPrefix(pkg, domain+"/") {
			return true
		}
	}
	return false
}

func isAdapter(pkg string) bool {
	return strings.HasPrefix(pkg, adapters)
}

func adapterName(pkg string) string {
	return strings.SplitN(strings.TrimPrefix(pkg, adapters), "/", 2)[0]
}

func isThirdParty(pkg string) bool {
	return pkg != module && !strings.HasPrefix(pkg, module+"/") && strings.Contains(strings.SplitN(pkg, "/", 2)[0], ".")
}

func isInfrastructure(pkg string) bool {
	root := strings.SplitN(pkg, "/", 2)[0]
	for _, forbidden := range infrastructure {
		if root == forbidden {
			return true
		}
	}
	return false
}

// TestArchitecture enforces the hexagonal dependency rule on production imports.
func TestArchitecture(t *testing.T) {
	output, err := exec.Command("go", "list", "-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}", "./...").Output()
	if err != nil {
		t.Fatalf("list packages: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		pkg, imports := fields[0], fields[1:]
		for _, imported := range imports {
			switch {
			case isDomain(pkg) && isAdapter(imported):
				t.Errorf("domain %s imports adapter %s", pkg, imported)
			case isDomain(pkg) && isThirdParty(imported):
				t.Errorf("domain %s imports framework %s", pkg, imported)
			case isDomain(pkg) && isInfrastructure(imported):
				t.Errorf("domain %s imports infrastructure %s", pkg, imported)
			case isAdapter(pkg) && isAdapter(imported) && adapterName(pkg) != adapterName(imported):
				t.Errorf("adapter %s imports another adapter %s", pkg, imported)
			case !isAdapter(pkg) && pkg != module && isAdapter(imported):
				t.Errorf("%s imports adapter %s; only the composition root (main.go) may construct adapters", pkg, imported)
			}
		}
	}
}
