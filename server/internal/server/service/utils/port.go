package utils

import (
	"errors"
	"strings"
)

func Port2Enterprise(port uint16) string {
	c := (A*int(port) + B) % M
	code5 := leftPad5(c)
	check := luhnDigit(code5)
	return code5 + string(rune('0'+check))
}

func Enterprise2Port(code6 string) (uint16, error) {
	if len(code6) != 6 || !isAllDigits(code6) {
		return 0, errors.New("企业号必须是6位数字")
	}
	if !verifyLuhn(code6) {
		return 0, errors.New("企业号校验失败")
	}
	c := atoi(code6[:5])
	p := (aInv * (c - B)) % M
	if p < 0 {
		p += M
	}
	if p > 65535 {
		return 0, errors.New("解码结果超出端口范围")
	}
	return uint16(p), nil
}

const M = 100000 // 10^5
const A = 70001
const B = 19731
const aInv = 30001 // 30001 = modInverse(70001, 100000)

// var aInv, _ = modInverse(A, M)

// 扩展欧几里得求逆元
func modInverse(a, m int) (int, error) {
	t, newT := 0, 1
	r, newR := m, a%m
	for newR != 0 {
		q := r / newR
		t, newT = newT, t-q*newT
		r, newR = newR, r-q*newR
	}
	if r != 1 {
		return 0, errors.New("not invertible")
	}
	if t < 0 {
		t += m
	}
	return t, nil
}

// ---- Luhn 校验 ----

func luhnDigit(code5 string) int {
	sum := 0
	double := true
	for i := len(code5) - 1; i >= 0; i-- {
		d := int(code5[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return (10 - (sum % 10)) % 10
}

func verifyLuhn(code6 string) bool {
	want := luhnDigit(code6[:5])
	return int(code6[5]-'0') == want
}

// ---- helpers ----
func leftPad5(n int) string {
	s := itoa(n)
	if len(s) >= 5 {
		return s
	}
	return strings.Repeat("0", 5-len(s)) + s
}
func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(buf[i:])
}
