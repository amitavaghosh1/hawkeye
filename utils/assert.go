package utils

import "log"

func CheckErr(err error, msg string) {
	if err == nil {
		return
	}

	log.Fatal(msg, err)

}
