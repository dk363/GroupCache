package main

import (
	"fmt"
	"golang.org/x/sync/singleflight"
	"time"
)

var callCount = 0

func getData() string {
	callCount++  // 每次调用就加1
	fmt.Printf("getData() 被调用了，这是第 %d 次调用\n", callCount)
	time.Sleep(2 * time.Second)
	return "hsu"
}

func main() {
	g := new(singleflight.Group)

	go func() {
		v1, _, shared := g.Do("getData", func()(interface{}, error) {
			ret := getData()
			return ret, nil
		})
		fmt.Printf("1st call: v1:%v, shared:%v\n", v1, shared)
	}()

	fmt.Print("1\n")
	// time.Sleep(2 * time.Second)
	// 如果取消注释这一行
	// shared: true

	v2, _, shared := g.Do("getData", func() (interface{}, error) {
		ret := getData()
		return ret, nil
	})
	fmt.Printf("2nd call: v2:%v, shared:%v\n", v2, shared)
	// time.Sleep(2 * time.Second)
}