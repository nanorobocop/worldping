package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/nanorobocop/worldping/task"

	"github.com/nanorobocop/worldping/db"
)

func parseUint(uintStr *string) (uintVar *uint32, intVar *int32, cidrVar string, err error) {
	uintVar64, err := strconv.ParseUint(*uintStr, 10, 32)
	if err != nil {
		return nil, nil, "", err
	}
	uintTmp := uint32(uintVar64)
	uintVar = &uintTmp

	intVar = db.UintToInt(*uintVar)
	cidrVar = task.IPToStr(*uintVar)
	return
}

func parseInt(intStr *string) (uintVar *uint32, intVar *int32, cidrVar string, er error) {
	intVar64, err := strconv.ParseInt(*intStr, 10, 32)
	if err != nil {
		return nil, nil, "", err
	}
	intTmp := int32(intVar64)
	intVar = &intTmp

	uintVar = db.IntToUint(*intVar)
	cidrVar = task.IPToStr(*uintVar)
	return
}

func parseCidr(cidrStr string) (uintVar *uint32, intVar *int32, cidrVar string, err error) {
	r := regexp.MustCompile(`^(?P\d{1,3})\.(?P\d{1,3})\.(?P\d{1,3})\.(?P\d{1,3})$`)
	matched := r.FindStringSubmatch(cidrStr)

	octet24, err := strconv.ParseUint(matched[1], 10, 32)
	if err != nil {
		return nil, nil, "", err
	}
	if octet24 >= 256 {
		return nil, nil, "", errors.New("First octet should be less 256")
	}

	octet16, err := strconv.ParseUint(matched[1], 10, 32)
	if err != nil {
		return nil, nil, "", err
	}
	if octet16 >= 256 {
		return nil, nil, "", errors.New("Second octet should be less 256")
	}

	octet8, err := strconv.ParseUint(matched[1], 10, 32)
	if err != nil {
		return nil, nil, "", err
	}
	if octet8 >= 256 {
		return nil, nil, "", errors.New("Third octet should be less 256")
	}

	octet0, err := strconv.ParseUint(matched[1], 10, 32)
	if err != nil {
		return nil, nil, "", err
	}
	if octet0 >= 256 {
		return nil, nil, "", errors.New("Fourth octet should be less 256")
	}
	uintTmp := uint32(octet24<<24 + octet16<<16 + octet8<<8 + octet0)
	uintVar = &uintTmp
	intVar = db.UintToInt(*uintVar)
	return
}

func main() {
	var uintVar *uint32
	var intVar *int32
	var cidrVar string
	var err error

	var uintStr = flag.String("uint", "", "uint representation of IP")
	var intStr = flag.String("int", "", "int representation of IP")
	var cidrStr = flag.String("cidr", "", "cidr representatino of IP")
	flag.Parse()

	if *uintStr != "" {
		uintVar, intVar, cidrVar, err = parseUint(uintStr)
	} else if *intStr != "" {
		uintVar, intVar, cidrVar, err = parseInt(intStr)
	} else if *cidrStr != "" {
		uintVar, intVar, cidrVar, err = parseInt(cidrStr)
	} else {
		fmt.Printf("Usage: %v [-int|-uint|-str] VALUE\n", os.Args[0])
		os.Exit(1)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Int: %d\n", intVar)
	fmt.Printf("Uint: %d\n", uintVar)
	fmt.Printf("Cidr: %s\n", cidrVar)
}
