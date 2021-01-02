package nineonesniffer

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
)

func decryptShit(shit1 string, shit2 string) (result string, e error) {
	cipher, e := ioutil.ReadAll(base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(shit1)))
	if e != nil {
		return "", e
	}

	key := []byte(shit2)

	var buf strings.Builder
	for i := 0; i < len(cipher); i++ {
		buf.WriteByte(cipher[i] ^ key[i%len(key)])
	}

	res, e := ioutil.ReadAll(base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(buf.String())))
	if e != nil {
		return "", e
	}

	return string(res), nil
}

func TryDecryptShit() {
	result, e := decryptShit("YC15QDQpGBozYDASUl43FggHQwkhJBheBg9xK2hrUAMAEhcTADUtTCFIakMHL1wZMgYeA3gAAAtlInxBCiRtEAEdMnY6chJtExJzACcdDxg/NxBoNBZceBtcWgolKQwwKEEkHhAJDlUJYQwsEjIlIQ5QWXsXIVMuG3FfBwgnAh0zW3cPK184HHsEWRsfQQMqHj0hISc0Mxg7A39BUx1ZCigiRDIiL102DiBYKwkUcQ4iAWsgJjoSAR82ZREWCi8gEg4EQwEgcx00FhIcEmksERINKC8EOnFYKGQALjoLRAZkDjJQeQ03MiBEARwvX3JdBzojbDYKZg8zUTIFOwwsPwFdfgV+JlQd", "0e76PqRpi3rh13z/B5+9ElYhJvHA1YdvOFQdNad9xzS7KVemQBOy4zQs+v58GMXdbYcYYACTpD9HjXYHmMB/yR7JUo6EvqVcrqjVJpT8Y5hFHeoojbl7ttfvVC84kqfXf6BJyobe4S5hNpADIPhWFlrbYW2q2JhsIL4fvB5cod4bkADLqxRLCWTiIL3AAMfWDtFpRJ5+PSXDH/bAEJzUQoAjK2Bfnf06+Vcb6XqetuGfz2B3NrqYUM36ybhuaKzIMoOr")

	if e != nil {
		fmt.Println(e)
	} else {
		fmt.Printf("result - %s\n", result)
	}
}
