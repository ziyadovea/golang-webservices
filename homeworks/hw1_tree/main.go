package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

// Функция отображения дерева каталогов
func dirTree(output io.Writer, path string, printFiles bool) error {
	levelPrint := make(map[int]bool)
	return dirTreeHelp(output, path, printFiles, 0, levelPrint)
}

// Вспомогательная функция фильтрации среза
func filter(files []os.FileInfo, test func(info os.FileInfo) bool) (res []os.FileInfo) {
	for _, file := range files {
		if test(file) {
			res = append(res, file)
		}
	}
	return
}

// Вспомогательная рекурсивная функция для dirTree
func dirTreeHelp(output io.Writer, path string, printFiles bool, level int, levelPrint map[int]bool) error {

	// Просмотр директории
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	// Сортировка по имени
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Фильтрация при условии, что нужен только вывод директорий
	if !printFiles {
		files = filter(files, func(info os.FileInfo) bool { return info.IsDir() })
	}
	count := len(files)

	// Цикл по всем директориям и/или файлам
	for ind, file := range files {

		// При последнем выводимом файле/директории не надо печатать вертикальный разделитель
		if ind == count-1 {
			levelPrint[level] = false
		}

		// Если файл является директорией
		if file.IsDir() {
			// Выводим нужное количество вертикальных разделителей и табуляцию
			for i := 0; i < level; i++ {
				if isPrint, ok := levelPrint[i]; ok && isPrint {
					fmt.Fprint(output, "│")
				}
				fmt.Fprint(output, "\t")
			}
			// Выводим необходимый элемент графики
			if len(files) > 1 && ind != count-1 {
				fmt.Fprint(output, "├───"+file.Name(), "\n")
				levelPrint[level] = true
			} else {
				fmt.Fprint(output, "└───"+file.Name(), "\n")
			}
			// Рекурсивно вызываем функцию уже в этой директории
			err = dirTreeHelp(output, path+string(os.PathSeparator)+file.Name(), printFiles, level+1, levelPrint)
			if err != nil {
				return err
			}
			// Обрабатываем файлы в директориях только при параметре printFiles == true
		} else if printFiles {
			// Находим размер файла
			var size string
			if file.Size() == 0 {
				size = " (empty)"
			} else {
				size = " (" + strconv.FormatInt(file.Size(), 10) + "b)"
			}
			// Выводим нужное количество вертикальных разделителей и табуляцию
			for i := 0; i < level; i++ {
				if isPrint, ok := levelPrint[i]; ok && isPrint {
					fmt.Fprint(output, "│")
				}
				fmt.Fprint(output, "\t")
			}
			// Выводим необходимый элемент графики
			if len(files) > 1 && ind != count-1 {
				fmt.Fprint(output, "├───"+file.Name()+size, "\n")
			} else {
				fmt.Fprint(output, "└───"+file.Name()+size, "\n")
			}
		}
	}
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
