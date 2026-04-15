package multipart_test

import (
	"bytes"
	"testing"

	"github.com/xbaseio/xbase/utils/xhttp/internal/multipart"
)

type User struct {
	ID   int    `json:"id" xml:"id"`
	Name string `json:"name" xml:"name"`
}

func TestWriter_WriteField(t *testing.T) {
	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)
	//user := &User{
	//	ID:   1,
	//	Name: "github.com/xbaseio/xbase",
	//}
	users := []*User{{
		ID:   1,
		Name: "github.com/xbaseio/xbase",
	}}

	err := writer.WriteField("users", users, multipart.FieldTypeXml)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(buffer.String())
}
