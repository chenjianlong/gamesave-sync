package gsutils

import "os"

const TimeFormat = "20060102150405"

// exists returns whether the given file or directory exists
func IsDir(path string) (bool, error) {
	st, err := os.Stat(path)
	if err == nil {
		return st.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
