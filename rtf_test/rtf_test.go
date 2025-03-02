package rtf_test

import (
   "bytes"
   "io/ioutil"
   "strings"
   "testing"

   "github.com/luisfurquim/docconv"
)

func TestConvertRTF(t *testing.T) {
   data, err := ioutil.ReadFile("testdata/test.rtf")
   if err != nil {
      t.Fatalf("got error %v, want nil", err)
   }
   res, _, err := docconv.ConvertRTF(bytes.NewReader(data))
   if err != nil {
      t.Fatalf("got error %v, want nil", err)
   }
   lines := strings.Split(res, "\n")[2:]
   line, expectedLine := lines[0], "hello world"
   if line != expectedLine {
      t.Fatalf("got %s, want %s", line, expectedLine)
   }
   line, expectedLine = lines[1], "helo"
   if line != expectedLine {
      t.Fatalf("got %s, want %s", line, expectedLine)
   }
   line, expectedLine = lines[2], ""
   if line != expectedLine {
      t.Fatalf("got %s, want %s", line, expectedLine)
   }
   line, expectedLine = lines[3], "1"
   if line != expectedLine {
      t.Fatalf("got %s, want %s", line, expectedLine)
   }
}
