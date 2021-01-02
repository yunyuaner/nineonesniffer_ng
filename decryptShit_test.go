package nineonesniffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	shit1     = "NC19FwABclwtGQ5VEEYOVBMFPFEVD3ZYdTh7TSIhBSEYXhZ+DlpRGCAEOEsOFGE9AS4eGQZMAAIrAmYBFjN8FlAAHngOBCkBdwpKRTFXEzsqNRtTHycccD06BEAHPwwmHTgYEDkLbSFoe3UIHhF5LTQGKT0iGWU3MwEAU1AQGUASejBJG3U4PRtEVScqJlBK"
	shit2     = "de3adY86wJL/s+CmY7TaqG7n9AC5mubTU4COB06alnS3BmXIbjOcJ6Mxex+3YpIb3DOWm7x88Ls/UdEaxXJ+PpVAnoI4HhBSSBpGlX7M8/09Pk8UyRpJls0YzIRf3WLyXIj9A2nKWvdP"
	decrypted = "http://198.255.82.91//mp43/337368.mp4?st=8_cwuYFd19buIC-9cn78VQ&e=1570116065"
)

/*
strencode("YC15QDQpGBozYDASUl43FggHQwkhJBheBg9xK2hrUAMAEhcTADUtTCFIakMHL1wZMgYeA3gAAAtlInxBCiRtEAEdMnY6chJtExJzACcdDxg/NxBoNBZceBtcWgolKQwwKEEkHhAJDlUJYQwsEjIlIQ5QWXsXIVMuG3FfBwgnAh0zW3cPK184HHsEWRsfQQMqHj0hISc0Mxg7A39BUx1ZCigiRDIiL102DiBYKwkUcQ4iAWsgJjoSAR82ZREWCi8gEg4EQwEgcx00FhIcEmksERINKC8EOnFYKGQALjoLRAZkDjJQeQ03MiBEARwvX3JdBzojbDYKZg8zUTIFOwwsPwFdfgV+JlQd","0e76PqRpi3rh13z/B5+9ElYhJvHA1YdvOFQdNad9xzS7KVemQBOy4zQs+v58GMXdbYcYYACTpD9HjXYHmMB/yR7JUo6EvqVcrqjVJpT8Y5hFHeoojbl7ttfvVC84kqfXf6BJyobe4S5hNpADIPhWFlrbYW2q2JhsIL4fvB5cod4bkADLqxRLCWTiIL3AAMfWDtFpRJ5+PSXDH/bAEJzUQoAjK2Bfnf06+Vcb6XqetuGfz2B3NrqYUM36ybhuaKzIMoOr","YC15QDQpGBozYDASUl43FggHQwkhJBheBg9xK2hrUAMAEhcTADUtTCFIakMHL1wZMgYeA3gAAAtlInxBCiRtEAEdMnY6chJtExJzACcdDxg/NxBoNBZceBtcWgolKQwwKEEkHhAJDlUJYQwsEjIlIQ5QWXsXIVMuG3FfBwgnAh0zW3cPK184HHsEWRsfQQMqHj0hISc0Mxg7A39BUx1ZCigiRDIiL102DiBYKwkUcQ4iAWsgJjoSAR82ZREWCi8gEg4EQwEgcx00FhIcEmksERINKC8EOnFYKGQALjoLRAZkDjJQeQ03MiBEARwvX3JdBzojbDYKZg8zUTIFOwwsPwFdfgV+JlQd1")
*/

func TestDecryptShit(t *testing.T) {
	result, e := decryptShit("YC15QDQpGBozYDASUl43FggHQwkhJBheBg9xK2hrUAMAEhcTADUtTCFIakMHL1wZMgYeA3gAAAtlInxBCiRtEAEdMnY6chJtExJzACcdDxg/NxBoNBZceBtcWgolKQwwKEEkHhAJDlUJYQwsEjIlIQ5QWXsXIVMuG3FfBwgnAh0zW3cPK184HHsEWRsfQQMqHj0hISc0Mxg7A39BUx1ZCigiRDIiL102DiBYKwkUcQ4iAWsgJjoSAR82ZREWCi8gEg4EQwEgcx00FhIcEmksERINKC8EOnFYKGQALjoLRAZkDjJQeQ03MiBEARwvX3JdBzojbDYKZg8zUTIFOwwsPwFdfgV+JlQd", "0e76PqRpi3rh13z/B5+9ElYhJvHA1YdvOFQdNad9xzS7KVemQBOy4zQs+v58GMXdbYcYYACTpD9HjXYHmMB/yR7JUo6EvqVcrqjVJpT8Y5hFHeoojbl7ttfvVC84kqfXf6BJyobe4S5hNpADIPhWFlrbYW2q2JhsIL4fvB5cod4bkADLqxRLCWTiIL3AAMfWDtFpRJ5+PSXDH/bAEJzUQoAjK2Bfnf06+Vcb6XqetuGfz2B3NrqYUM36ybhuaKzIMoOr")
	assert.Nil(t, e, "decryption should be okay")
	assert.Contains(t, result, `<source src="https://ccm.91p52.com/358999.mp4?st=6dX3e0ya0njWeGIALSBPrA&f=ed08LJ7jxrtD3siJbLVA5QnVBHx2NHF/Y937gfcALELYsZQ4OxoLTI9HNGk6K7pCiKaORleJhB9CrfSSFOWPGqrXYkhVHQe+p/xp" type='video/mp4'>`)
}