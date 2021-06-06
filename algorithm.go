package nineonesniffer

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func decode(infoStr string) (string, string) {
	start := strings.Index(infoStr, "\"") + 1
	end := strings.LastIndex(infoStr, "\"")

	escapedSrc := infoStr[start:end]

	var b bytes.Buffer

	for where := 0; where < len(escapedSrc); where += 3 {
		n := strings.Index(escapedSrc[where:], "%")
		val := escapedSrc[where+n+1 : where+n+3]
		integerCh, _ := strconv.ParseInt(val, 16, 32)
		b.WriteByte(byte(integerCh))
	}

	/**
	 * unescaped may looks like:
	 * - Case 1)
	 * <source src='https://ccn.91p52.com//m3u8/459666/459666.m3u8?st=TM6j903f8X4G4lu2lkxyMQ&e=1619197640' type='application/x-mpegURL'>
	 * - Case 2)
	 * <source src='https://fdc.91p49.com/m3u8/459666/459666.m3u8' type='application/x-mpegURL'>
	 * notice that the former url doesn't have http get parameters!!
	 */
	unescaped := b.String()

	fmt.Println(unescaped)

	start = strings.Index(unescaped, "src='") + len("src='")
	end = strings.Index(unescaped[start:], "'")
	srcWithParams := unescaped[start : start+end]
	questionMarkPos := strings.Index(srcWithParams, "?")
	var name string

	fmt.Println(srcWithParams)

	if questionMarkPos == -1 {
		/* Case 2), in case of no http get parameters */
		slash := strings.LastIndex(srcWithParams, "/")
		name = srcWithParams[slash+1:]
	} else {
		/* Case 1) */
		httpGetSrc := srcWithParams[:questionMarkPos]
		slash := strings.LastIndex(httpGetSrc, "/")
		name = httpGetSrc[slash+1:]
	}

	return name, srcWithParams
}
