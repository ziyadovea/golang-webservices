package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func FastSearch(out io.Writer) {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fmt.Fprintln(out, "found users:")

	browsers := make([]string, 0, 1000)
	i := -1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		i++

		user := User{}
		err := user.UnmarshalJSON(scanner.Bytes())
		if err != nil {
			panic(err)

		}

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {

			if strings.Contains(browser, "Android") {
				isAndroid = true
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
			} else {
				continue
			}

			if !Contains(browsers, browser) {
				browsers = append(browsers, browser)
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := strings.ReplaceAll(user.Email, "@", " [at] ")
		fmt.Fprintln(out, fmt.Sprintf("[%d] %s <%s>", i, user.Name, email))

	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(browsers))
}
