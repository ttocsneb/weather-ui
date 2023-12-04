package server

import (
	"fmt"
	"io/fs"
)

func ReadDirRecursive(fsys fs.FS, name string) ([]string, error) {
	ents, err := fs.ReadDir(fsys, name)
	if err != nil {
		return nil, err
	}

	result := []string{}

	for _, ent := range ents {
		ent_name := ent.Name()
		if ent.Type().IsDir() {
			n := fmt.Sprintf("%v/%v", name, ent_name)
			files, err := ReadDirRecursive(fsys, n)
			if err == fs.ErrPermission {
				continue
			}
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				result = append(result, f)
			}
		} else {
			result = append(result, fmt.Sprintf("%v/%v", name, ent_name))
		}
	}
	return result, nil
}
