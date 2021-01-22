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

const (
	encrypted = "%3c%73%6f%75%72%63%65%20%73%72%63%3d%27%68%74%74%70%73%3a%2f%2f%63%63%6e%2e%39%31%70%35%32%2e%63%6f%6d%2f%2f%6d%33%75%38%2f%34%32%34%30%38%30%2f%34%32%34%30%38%30%2e%6d%33%75%38%3f%73%74%3d%68%47%42%70%43%57%55%5a%44%49%61%4e%6c%4f%46%47%2d%43%52%6f%65%41%26%65%3d%31%36%31%30%32%37%37%34%31%33%26%66%3d%33%38%62%61%55%4e%2b%73%45%69%48%49%69%45%36%57%4e%56%5a%42%34%58%7a%30%43%42%48%71%36%58%72%33%42%72%78%64%58%73%63%48%48%2b%52%41%57%45%6c%61%6d%61%6b%6d%78%73%54%6a%55%6a%70%33%2b%6a%76%59%52%5a%2f%55%2b%41%55%36%37%44%69%52%55%66%5a%6d%48%72%4e%46%4f%63%75%31%61%31%55%33%72%59%67%37%53%4f%36%6b%6f%65%59%70%2f%34%2f%62%42%31%4d%6e%57%41%27%20%74%79%70%65%3d%27%61%70%70%6c%69%63%61%74%69%6f%6e%2f%78%2d%6d%70%65%67%55%52%4c%27%3e"

	decrypted = "<source src='https://ccn.91p52.com//m3u8/424080/424080.m3u8?st=hGBpCWUZDIaNlOFG-CRoeA&e=1610277413&f=38baUN+sEiHIiE6WNVZB4Xz0CBHq6Xr3BrxdXscHH+RAWElamakmxsTjUjp3+jvYRZ/U+AU67DiRUfZmHrNFOcu1a1U3rYg7SO6koeYp/4/bB1MnWA' type='application/x-mpegURL'>"
)

func TryDecryptShit() {
	result, e := decryptShit("YC15QDQpGBozYDASUl43FggHQwkhJBheBg9xK2hrUAMAEhcTADUtTCFIakMHL1wZMgYeA3gAAAtlInxBCiRtEAEdMnY6chJtExJzACcdDxg/NxBoNBZceBtcWgolKQwwKEEkHhAJDlUJYQwsEjIlIQ5QWXsXIVMuG3FfBwgnAh0zW3cPK184HHsEWRsfQQMqHj0hISc0Mxg7A39BUx1ZCigiRDIiL102DiBYKwkUcQ4iAWsgJjoSAR82ZREWCi8gEg4EQwEgcx00FhIcEmksERINKC8EOnFYKGQALjoLRAZkDjJQeQ03MiBEARwvX3JdBzojbDYKZg8zUTIFOwwsPwFdfgV+JlQd", "0e76PqRpi3rh13z/B5+9ElYhJvHA1YdvOFQdNad9xzS7KVemQBOy4zQs+v58GMXdbYcYYACTpD9HjXYHmMB/yR7JUo6EvqVcrqjVJpT8Y5hFHeoojbl7ttfvVC84kqfXf6BJyobe4S5hNpADIPhWFlrbYW2q2JhsIL4fvB5cod4bkADLqxRLCWTiIL3AAMfWDtFpRJ5+PSXDH/bAEJzUQoAjK2Bfnf06+Vcb6XqetuGfz2B3NrqYUM36ybhuaKzIMoOr")

	if e != nil {
		fmt.Println(e)
	} else {
		fmt.Printf("result - %s\n", result)
	}
}
