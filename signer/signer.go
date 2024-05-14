package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// сюда писать код
func main() {
	testResult := "NOT_SET"

	// это небольшая защита от попыток не вызывать мои функции расчета
	// я преопределяю фукции на свои которые инкрементят локальный счетчик
	// переопределение возможо потому что я объявил функцию как переменную, в которой лежит функция

	// This is a small check to verify that you are actually using supplied `DataSignerMd5` and
	// `DataSignerCrc32` functions. These function are substituted by the ones that are incrementing
	// some local counter. Substitution is possible due to the fact that functions are passed as
	// variables.

	var (
		DataSignerSalt         string = "" // на сервере будет другое значение
		OverheatLockCounter    uint32
		OverheatUnlockCounter  uint32
		DataSignerMd5Counter   uint32
		DataSignerCrc32Counter uint32
	)
	OverheatLock = func() {
		atomic.AddUint32(&OverheatLockCounter, 1)
		for {
			if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 0, 1); !swapped {
				fmt.Println("OverheatLock happend")
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}
	OverheatUnlock = func() {
		atomic.AddUint32(&OverheatUnlockCounter, 1)
		for {
			if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 1, 0); !swapped {
				fmt.Println("OverheatUnlock happend")
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}
	DataSignerMd5 = func(data string) string {
		atomic.AddUint32(&DataSignerMd5Counter, 1)
		OverheatLock()
		defer OverheatUnlock()
		data += DataSignerSalt
		dataHash := fmt.Sprintf("%x", md5.Sum([]byte(data)))
		time.Sleep(10 * time.Millisecond)
		return dataHash
	}
	DataSignerCrc32 = func(data string) string {
		atomic.AddUint32(&DataSignerCrc32Counter, 1)
		data += DataSignerSalt
		crcH := crc32.ChecksumIEEE([]byte(data))
		dataHash := strconv.FormatUint(uint64(crcH), 10)
		time.Sleep(time.Second)
		return dataHash
	}

	//inputData := []int{0, 1, 1, 2, 3, 5, 8}
	inputData := []int{0, 1}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
			}
			testResult = data

		}),
	}

	start := time.Now()

	ExecutePipeline(hashSignJobs...)

	end := time.Since(start)
	fmt.Println("time:", end)
	fmt.Println("Result:", testResult)
}

func ExecutePipeline(jobs ...job) {

	var channels []chan interface{} = make([]chan interface{}, len(jobs)+1)
	for i := 0; i < len(channels); i++ {
		channels[i] = make(chan interface{})
	}

	for i := 0; i < len(jobs); i++ {
		go func(index int) {
			jobs[index](channels[index], channels[index+1])
			if index == 0 {
				close(channels[0])
			}
			close(channels[index+1])
		}(i)
	}

	for _ = range channels[len(jobs)] {

	}
}

func SingleHash(in, out chan interface{}) {
	var countGorutine int
	muCount := &sync.Mutex{}

	muMd5 := &sync.Mutex{}
	for v := range in {
		go func(value string) {
			calcOneValueSingleHash(value, muMd5, out)
			muCount.Lock()
			countGorutine--
			muCount.Unlock()
		}(fmt.Sprint(v.(int)))

		muCount.Lock()
		countGorutine++
		muCount.Unlock()
	}

	isZero := func() bool {
		muCount.Lock()
		defer muCount.Unlock()
		return countGorutine == 0
	}
	for !isZero() {

	}
}

func calcOneValueSingleHash(value string, muMd5 *sync.Mutex, out chan interface{}) {
	var hashResult1 = make(chan string)
	go func() {
		hashResult1 <- DataSignerCrc32(value)
	}()

	var hashResult2 = make(chan string)
	go func() {
		muMd5.Lock()
		var dataMd5 = DataSignerMd5(value)
		muMd5.Unlock()
		hashResult2 <- DataSignerCrc32(dataMd5)
	}()

	var stringBuilder strings.Builder = strings.Builder{}
	stringBuilder.WriteString(<-hashResult1)
	stringBuilder.WriteString("~")
	stringBuilder.WriteString(<-hashResult2)
	out <- stringBuilder.String()
}

func MultiHash(in, out chan interface{}) {
	var countGorutine int
	muCount := &sync.Mutex{}

	for v := range in {
		go func() {
			calcOneValueMultiHash(v.(string), out)
			muCount.Lock()
			countGorutine--
			muCount.Unlock()
		}()

		muCount.Lock()
		countGorutine++
		muCount.Unlock()
	}

	isZero := func() bool {
		muCount.Lock()
		defer muCount.Unlock()
		return countGorutine == 0
	}
	for !isZero() {

	}
}

func calcOneValueMultiHash(value string, out chan interface{}) {
	var sliceHash = make([]string, 6)
	var wg sync.WaitGroup
	wg.Add(6)
	for th := 0; th < 6; th++ {
		go func(inParam string, number int) {
			defer wg.Done()
			sliceHash[number] = DataSignerCrc32(inParam)
		}(fmt.Sprintf("%d%s", th, value), th)
	}
	wg.Wait()
	out <- strings.Join(sliceHash, "")
}

func CombineResults(in, out chan interface{}) {
	resultSet := make([]string, 0)
	for v := range in {
		resultSet = append(resultSet, v.(string))
	}
	sort.Slice(resultSet, func(i, j int) bool {
		return resultSet[i] < resultSet[j]
	})
	out <- strings.Join(resultSet, "_")
}
