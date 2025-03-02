package docconv

import (
   "bytes"
   "encoding/xml"
   "fmt"
   "io"
)

// ConvertXML converts an XML file to text.
func ConvertXML(r io.Reader) (string, map[string]string, error) {
   meta := make(map[string]string)
   cleanXML, err := Tidy(r, true)
   if err != nil {
      return "", nil, fmt.Errorf("tidy error: %v", err)
   }
   result, err := XMLToText(bytes.NewReader(cleanXML), []string{}, []string{}, true)
   if err != nil {
      return "", nil, fmt.Errorf("error from XMLToText: %v", err)
   }
   return result, meta, nil
}

// ConvertXMLParagraphs converts an XML file to text divided in paragraphs.
func ConvertXMLParagraphs(r io.Reader) ([]string, error) {
   var par []string

   cleanXML, err := Tidy(r, true)
   if err != nil {
      return nil, fmt.Errorf("tidy error: %v", err)
   }
   par, err = XMLToTextParagraphs(bytes.NewReader(cleanXML), []string{}, []string{}, []string{}, []string{}, true)
   if err != nil {
      return nil, fmt.Errorf("error from XMLToText: %v", err)
   }

   return par, nil
}

// XMLToText converts XML to plain text given how to treat elements.
func XMLToText(r io.Reader, breaks []string, skip []string, strict bool) (string, error) {
   var result string

   dec := xml.NewDecoder(r)
   dec.Strict = strict
   for {
      t, err := dec.Token()
      if err != nil {
         if err == io.EOF {
            break
         }
         return "", err
      }

      switch v := t.(type) {
      case xml.CharData:
         result += string(v)
      case xml.StartElement:
         for _, breakElement := range breaks {
            if v.Name.Local == breakElement {
               result += "\n"
            }
         }
         for _, skipElement := range skip {
            if v.Name.Local == skipElement {
               depth := 1
               for {
                  t, err := dec.Token()
                  if err != nil {
                     // An io.EOF here is actually an error.
                     return "", err
                  }

                  switch t.(type) {
                  case xml.StartElement:
                     depth++
                  case xml.EndElement:
                     depth--
                  }

                  if depth == 0 {
                     break
                  }
               }
            }
         }
      }
   }
   return result, nil
}

// XMLToTextParagraphs converts XML to plain text divided in paragraphs given how to treat elements.
func XMLToTextParagraphs(r io.Reader, content, grabs, breaks []string, skip []string, strict bool) ([]string, error) {
   var result []string
   var par int
   var grab bool

   par = -1
   dec := xml.NewDecoder(r)
   dec.Strict = strict
   for {
      t, err := dec.Token()
      if err != nil {
         if err == io.EOF {
            break
         }
         return nil, err
      }

      switch v := t.(type) {
      case xml.CharData:
         if grab {
            result[par] += string(v)
         }
      case xml.StartElement:
         grab = false
         for _, contentElement := range content {
            if v.Name.Local == contentElement {
               par = len(result)
               result = append(result, "")
               break
            }
         }
         if len(grabs) == 0 {
            grab = true
         }
         for _, grabElement := range grabs {
            if v.Name.Local == grabElement {
               grab = true
               break
            }
         }
         for _, breakElement := range breaks {
            if v.Name.Local == breakElement {
               result[par] += "\n"
               break
            }
         }
         for _, skipElement := range skip {
            if v.Name.Local == skipElement {
               depth := 1
               for {
                  t, err := dec.Token()
                  if err != nil {
                     // An io.EOF here is actually an error.
                     return nil, err
                  }

                  switch t.(type) {
                  case xml.StartElement:
                     depth++
                  case xml.EndElement:
                     depth--
                  }

                  if depth == 0 {
                     break
                  }
               }
               break
            }
         }
      }
   }
   return result, nil
}

// XMLToMap converts XML to a nested string map.
func XMLToMap(r io.Reader) (map[string]string, error) {
   m := make(map[string]string)
   dec := xml.NewDecoder(r)
   var tagName string
   for {
      t, err := dec.Token()
      if err != nil {
         if err == io.EOF {
            break
         }
         return nil, err
      }

      switch v := t.(type) {
      case xml.StartElement:
         tagName = string(v.Name.Local)
      case xml.CharData:
         m[tagName] = string(v)
      }
   }
   return m, nil
}
