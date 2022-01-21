package main

import (
	"strings"
)

func trimLambdaSuffix(s string) string {
	end := len(s) - 1
	for {
		i := end
		for s[i] >= '0' && s[i] <= '9' {
			i--
		}
		found := false
		if strings.HasSuffix(s[:i+1], ".func") {
			i -= len(".func")
			found = true
		} else if s[i] == '.' {
			i--
			found = true
		}
		if !found {
			break
		}
		end = i
	}
	return s[:end+1]
}

func parseFuncName(s string) (pkgName, typeName, funcName string) {
	lastSlash := strings.LastIndexByte(s, '/')
	if lastSlash != -1 {
		s = s[lastSlash+len("/"):]
	}

	i := strings.IndexByte(s, '.')
	if i == -1 {
		return "", "", s
	}
	resultPkgName := s[:i]
	rest := s[i+1:]
	if strings.HasPrefix(rest, "(") {
		offset := 1
		if strings.HasPrefix(rest, "(*") {
			offset++
		}
		rparen := strings.IndexByte(rest, ')')
		if rparen == -1 {
			return "", "", ""
		}
		resultTypeName := rest[offset:rparen]
		resultFuncName := rest[rparen+len(")."):]
		return resultPkgName, resultTypeName, trimLambdaSuffix(resultFuncName)
	}
	return resultPkgName, "", trimLambdaSuffix(rest)
}
