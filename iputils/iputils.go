package iputils

import (
	"errors"
	"strconv"
	"strings"
)

type IP [4]byte

func IPStringToIPBytes(IPString string) (IP, error) {
	var IPBlockBuilder strings.Builder
	var IP4Bytes []byte
	if len(IPString) < 7 || len(IPString) > 15 {
		return IP{}, errors.New("invalid IPV4 adress length")
	}
	IPStrSplitted := strings.Split(IPString, "")
	for i := 0; i < len(IPStrSplitted); i++ {
		if _, err := strconv.Atoi(IPStrSplitted[i]); err != nil && IPStrSplitted[i] != "." {
			return IP{}, errors.New("IP must contain only dots and numbers")
		}
		if len(IPBlockBuilder.String()) > 3 {
			return IP{}, errors.New("one of the octets in the IP is too big")
		}
		if IPStrSplitted[i] != "." {
			_, err := IPBlockBuilder.WriteString(IPStrSplitted[i])
			if err != nil {
				return IP{}, err
			}
		}
		if IPStrSplitted[i] == "." || i == len(IPStrSplitted)-1 {
			IPBlockInt, _ := strconv.Atoi(IPBlockBuilder.String())
			IP4Bytes = append(IP4Bytes, byte(IPBlockInt))
			IPBlockBuilder.Reset()
			continue
		}
	}
	return IP(IP4Bytes), nil
}
