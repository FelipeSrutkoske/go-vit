package filesystem

import (
	"fmt"
	"os/exec"
)

func GetCurrentBranch() string {
	curBranch, err := exec.Command("git", "branch").Output()
	if err != nil {
		fmt.Println("Erro ao buscar branch.")
	}

	return string(curBranch)
}
