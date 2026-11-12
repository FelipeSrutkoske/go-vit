package filesystem

import "os"

func GetCurDir() string {
	curDir, err := os.Getwd()
	if err != nil {
		return "Falha ao obter o diretorio"
	}

	return string(curDir)
}
