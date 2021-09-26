package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// "Вы можете ожидать, что у вас никогда не будет более 100 элементов во входных данных" ©
const maxCountOfElems = 100

// ExecutePipeline реализует конвейер
func ExecutePipeline(jobs ...job) {

	// создаем каналы для передачи в функцию job
	var in chan interface{}
	var out chan interface{}
	// создаем объект типа sync.WaitGroup для синхронизации горутин
	var wg sync.WaitGroup // wg := &sync.WaitGroup{} - равносильно

	// цикл по всем задачам
	for _, j := range jobs {
		// на вход текущей задачи посылаем выход предыдущей задачи
		in = out
		// создаем выходной буферизированный канал
		out = make(chan interface{}, maxCountOfElems)
		// инкременнт wg
		wg.Add(1)
		// запускаем задачи в отдельных горутинах
		// параметры нужны, тк мы не знаем, в какой именно момент начнет выполняться горутина (вероятно, после конца цикла)
		go func(in, out chan interface{}, j job) {
			// отложенный декремент wg
			defer wg.Done()
			// отложенное закрытие выходного канала
			defer close(out)
			// запускаем очередной воркер
			j(in, out)
		}(in, out, j)
	}

	// ждем, пока wg станет == 0
	wg.Wait()

}

// ---

// CalculateHash1 считает значение выражения crc32(data)+"~"+crc32(md5(data))
func CalculateHash1(mu *sync.Mutex, val string) string {

	hash1 := make(chan string)
	hash2 := make(chan string)

	go func(hash1 chan string) {
		defer close(hash1)
		hash1 <- DataSignerCrc32(val)
	}(hash1)

	go func(hash2 chan string) {
		defer close(hash2)
		mu.Lock()
		md5 := DataSignerMd5(val)
		mu.Unlock()
		hash2 <- DataSignerCrc32(md5)
	}(hash2)

	return <-hash1 + "~" + <-hash2
}

// SingleHash считает значение crc32(data)+"~"+crc32(md5(data)) (конкатенация двух строк через ~),
// где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in, out chan interface{}) {

	// Создаем необходимые объекты для корректной работы с многопоточностью
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	// Цикл по всем значениям в канале
	for val := range in {
		wg.Add(1)
		// Параллельно обрабатываем данные
		go func(out chan interface{}, val string, mu *sync.Mutex) {
			defer wg.Done()
			// Считаем значение хэша
			out <- CalculateHash1(mu, val)
		}(out, fmt.Sprintf("%v", val), mu)
	}
	wg.Wait()
}

// CalculateHash2 считает значение выражения crc32(th+data) (конкатенация цифры, приведённой к строке и строки), th=0..5
func CalculateHash2(val string) string {

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	threadHash := make(map[int]string, 6)

	for th := 0; th < 6; th++ {
		wg.Add(1)
		go func(th int) {
			defer wg.Done()
			hash := DataSignerCrc32(fmt.Sprintf("%v", th) + val)
			// мапа в го - конкуретно небезопасный тип данных, поэтому используем mutex
			mu.Lock()
			threadHash[th] = hash
			mu.Unlock()
		}(th)
	}
	wg.Wait()

	res := ""
	for th := 0; th < 6 ; th++ {
		res += threadHash[th]
	}

	return res
}

// MultiHash считает значение crc32(th+data) (конкатенация цифры, приведённой к строке и строки),
// где th=0..5 (т.е. 6 хешей на каждое входящее значение), потом берёт конкатенацию результатов
// в порядке расчета (0..5), где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for val := range in {
		wg.Add(1)
		go func(out chan interface{}, val string) {
			defer wg.Done()
			out <- CalculateHash2(val)
		}(out, fmt.Sprintf("%v", val))
	}
	wg.Wait()
}

// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/), объединяет
// отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in, out chan interface{}) {
	res := make([]string, 0)
	for val := range in {
		res = append(res, fmt.Sprintf("%v", val))
	}
	sort.Strings(res)
	out <- strings.Join(res, "_")
}
