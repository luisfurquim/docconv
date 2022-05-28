package docconv

import (
   "archive/zip"
   "bytes"
   "encoding/xml"
   "fmt"
   "io"
   "io/ioutil"
   "os"
   "time"
)

type typeOverride struct {
   XMLName     xml.Name `xml:"Override"`
   ContentType string   `xml:"ContentType,attr"`
   PartName    string   `xml:"PartName,attr"`
}

type contentTypeDefinition struct {
   XMLName   xml.Name       `xml:"Types"`
   Overrides []typeOverride `xml:"Override"`
}

// ConvertDocx converts an MS Word docx file to text.
func ConvertDocx(r io.Reader) (string, map[string]string, error) {
   var size int64

   // Common case: if the reader is a file (or trivial wrapper), avoid
   // loading it all into memory.
   var ra io.ReaderAt
   if f, ok := r.(interface {
      io.ReaderAt
      Stat() (os.FileInfo, error)
   }); ok {
      si, err := f.Stat()
      if err != nil {
         return "", nil, err
      }
      size = si.Size()
      ra = f
   } else {
      b, err := ioutil.ReadAll(r)
      if err != nil {
         return "", nil, nil
      }
      size = int64(len(b))
      ra = bytes.NewReader(b)
   }

   zr, err := zip.NewReader(ra, size)
   if err != nil {
      return "", nil, fmt.Errorf("error unzipping data: %v", err)
   }

   zipFiles := mapZipFiles(zr.File)

   contentTypeDefinition, err := getContentTypeDefinition(zipFiles["[Content_Types].xml"])
   if err != nil {
      return "", nil, err
   }

   meta := make(map[string]string)
   var textHeader, textBody, textFooter string
   for _, override := range contentTypeDefinition.Overrides {
      f := zipFiles[override.PartName]

      switch {
      case override.ContentType == "application/vnd.openxmlformats-package.core-properties+xml":
         rc, err := f.Open()
         if err != nil {
            return "", nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
         }
         defer rc.Close()

         meta, err = XMLToMap(rc)
         if err != nil {
            return "", nil, fmt.Errorf("error parsing '%v': %v", f.Name, err)
         }

         if tmp, ok := meta["modified"]; ok {
            if t, err := time.Parse(time.RFC3339, tmp); err == nil {
               meta["ModifiedDate"] = fmt.Sprintf("%d", t.Unix())
            }
         }
         if tmp, ok := meta["created"]; ok {
            if t, err := time.Parse(time.RFC3339, tmp); err == nil {
               meta["CreatedDate"] = fmt.Sprintf("%d", t.Unix())
            }
         }
      case override.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml":
         body, err := parseDocxText(f)
         if err != nil {
            return "", nil, err
         }
         textBody += body + "\n"
      case override.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml":
         footer, err := parseDocxText(f)
         if err != nil {
            return "", nil, err
         }
         textFooter += footer + "\n"
      case override.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml":
         header, err := parseDocxText(f)
         if err != nil {
            return "", nil, err
         }
         textHeader += header + "\n"
      }

   }
   return textHeader + "\n" + textBody + "\n" + textFooter, meta, nil
}


// ConvertDocx converts an MS Word docx file to text.
func ConvertDocxParagraphs(r io.Reader) (*Paragraphs, error) {
   var size int64
   var par Paragraphs
   var err error

   // Common case: if the reader is a file (or trivial wrapper), avoid
   // loading it all into memory.
   var ra io.ReaderAt
   if f, ok := r.(interface {
      io.ReaderAt
      Stat() (os.FileInfo, error)
   }); ok {
      si, err := f.Stat()
      if err != nil {
         return nil, err
      }
      size = si.Size()
      ra = f
   } else {
      b, err := ioutil.ReadAll(r)
      if err != nil {
         return nil, nil
      }
      size = int64(len(b))
      ra = bytes.NewReader(b)
   }

   zr, err := zip.NewReader(ra, size)
   if err != nil {
      return nil, fmt.Errorf("error unzipping data: %v", err)
   }

   zipFiles := mapZipFiles(zr.File)

   contentTypeDefinition, err := getContentTypeDefinition(zipFiles["[Content_Types].xml"])
   if err != nil {
      return nil, err
   }

   for _, override := range contentTypeDefinition.Overrides {
      f := zipFiles[override.PartName]

      switch {
      case override.ContentType == "application/vnd.openxmlformats-package.core-properties+xml":
         rc, err := f.Open()
         if err != nil {
            return nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
         }
         defer rc.Close()

         par.Meta, err = XMLToMap(rc)
         if err != nil {
            return nil, fmt.Errorf("error parsing '%v': %v", f.Name, err)
         }

         if tmp, ok := par.Meta["modified"]; ok {
            if t, err := time.Parse(time.RFC3339, tmp); err == nil {
               par.Meta["ModifiedDate"] = fmt.Sprintf("%d", t.Unix())
            }
         }
         if tmp, ok := par.Meta["created"]; ok {
            if t, err := time.Parse(time.RFC3339, tmp); err == nil {
               par.Meta["CreatedDate"] = fmt.Sprintf("%d", t.Unix())
            }
         }
      case override.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml":
         body, err := parseDocxTextParagraphs(f)
         if err != nil {
            return nil, err
         }
         par.Body = append(par.Body, body...)
      case override.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml":
         footer, err := parseDocxTextParagraphs(f)
         if err != nil {
            return nil, err
         }
         par.Footer = append(par.Footer, footer...)
      case override.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml":
         header, err := parseDocxTextParagraphs(f)
         if err != nil {
            return nil, err
         }
         par.Header = append(par.Header, header...)
      }

   }
   return &par, nil
}




func getContentTypeDefinition(zf *zip.File) (*contentTypeDefinition, error) {
   f, err := zf.Open()
   if err != nil {
      return nil, err
   }
   defer f.Close()

   x := &contentTypeDefinition{}
   if err := xml.NewDecoder(f).Decode(x); err != nil {
      return nil, err
   }
   return x, nil
}

func mapZipFiles(files []*zip.File) map[string]*zip.File {
   filesMap := make(map[string]*zip.File, 2*len(files))
   for _, f := range files {
      filesMap[f.Name] = f
      filesMap["/"+f.Name] = f
   }
   return filesMap
}

func parseDocxText(f *zip.File) (string, error) {
   r, err := f.Open()
   if err != nil {
      return "", fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
   }
   defer r.Close()

   text, err := DocxXMLToText(r)
   if err != nil {
      return "", fmt.Errorf("error parsing '%v': %v", f.Name, err)
   }
   return text, nil
}


func parseDocxTextParagraphs(f *zip.File) ([]string, error) {
   r, err := f.Open()
   if err != nil {
      return nil, fmt.Errorf("error opening '%v' from archive: %v", f.Name, err)
   }
   defer r.Close()

   text, err := DocxXMLToTextParagraphs(r)
   if err != nil {
      return nil, fmt.Errorf("error parsing '%v': %v", f.Name, err)
   }
   return text, nil
}

// DocxXMLToText converts Docx XML into plain text.
func DocxXMLToText(r io.Reader) (string, error) {
   return XMLToText(r, []string{"br", "p", "tab"}, []string{"instrText", "script"}, true)
}

func DocxXMLToTextParagraphs(r io.Reader) ([]string, error) {
   return XMLToTextParagraphs(r, []string{"p"}, []string{"t"}, []string{"br", "tab"}, []string{"instrText", "script"}, true)
}
