package frames

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"
)

// WriteCSV writes: No,Time,Kind,Source,Identifier,Frequency,RSSI,Info.
func WriteCSV(w io.Writer, fs []Frame) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"No", "Time", "Kind", "Source", "Identifier", "Frequency", "RSSI", "Info"}); err != nil {
		return err
	}
	for _, f := range fs {
		rssi, freq := "", ""
		if f.RSSI != nil {
			rssi = strconv.Itoa(*f.RSSI)
		}
		if f.Freq != 0 {
			freq = strconv.FormatFloat(f.Freq, 'f', -1, 64)
		}
		row := []string{
			strconv.FormatUint(f.ID, 10), f.Time.Format(time.RFC3339Nano), f.Kind, f.Source, f.Identifier, freq, rssi, f.Info,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
