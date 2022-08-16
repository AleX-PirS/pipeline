package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

var muMd5 = &sync.Mutex{}

func ExecutePipeline(fun ...job) {
	channels := make([]chan interface{}, len(fun)+1)
	channels[0] = make(chan interface{})
	for idx, foo := range fun {
		channels[idx+1] = make(chan interface{})

		go func(idx int, foo job) {
			foo(channels[idx], channels[idx+1])
			close(channels[idx+1])
		}(idx, foo)
	}

	for elem := range channels[len(channels)-1] {
		fmt.Println(elem)
	}
}

func SingleHash(in, out chan interface{}) {
	var counter int32
	for val := range in {
		atomic.AddInt32(&counter, 1)

		go func(val interface{}) {
			dataIn := strconv.Itoa(val.(int))

			cr1 := make(chan interface{})
			cr2 := make(chan interface{})
			md := make(chan interface{})

			go func() {
				muMd5.Lock()
				md <- DataSignerMd5(dataIn)
				muMd5.Unlock()
			}()

			go func() {
				cr1 <- DataSignerCrc32(dataIn)
			}()

			go func() {
				cr2 <- DataSignerCrc32((<-md).(string))
			}()

			out <- (<-cr1).(string) + "~" + (<-cr2).(string)
			atomic.AddInt32(&counter, -1)
		}(val)
	}

	for atomic.LoadInt32(&counter) != 0 {
	}
}

func MultiHash(in, out chan interface{}) {
	var counter int32
	for val := range in {
		atomic.AddInt32(&counter, 1)

		go func(val interface{}) {
			dataIn := val.(string)
			chanSlice := make([]chan string, 6)

			for th := 0; th < 6; th++ {
				ch := make(chan string)
				chanSlice[th] = ch
				go func(th int) {
					chanSlice[th] <- DataSignerCrc32(strconv.Itoa(th) + dataIn)
				}(th)
			}

			var result string
			for th := 0; th < 6; th++ {
				result += <-chanSlice[th]
			}

			out <- result
			atomic.AddInt32(&counter, -1)
		}(val)
	}

	for atomic.LoadInt32(&counter) != 0 {
	}
}

func CombineResults(in, out chan interface{}) {
	dataSlice := make([]string, 0)
	for val := range in {
		dataSlice = append(dataSlice, val.(string))
	}
	sort.Strings(dataSlice)

	var result string
	for _, elem := range dataSlice {
		result += elem + "_"
	}

	out <- result[:len(result)-1]
}
