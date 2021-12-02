package bytes_test

import (
	"testing"

	"codeberg.org/gruf/go-bytes"
)

func ensurePanic(do func()) bool {
	var r interface{}
	func() {
		defer func() {
			r = recover()
		}()
		do()
	}()
	return r != nil
}

func TestWrite(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.Write([]byte("string"))
	if b.StringPtr() != "string" {
		t.Fatalf("Expected '%s', got '%s'", "string", b.StringPtr())
	}
	b.Reset()

	b.WriteString("string")
	if b.StringPtr() != "string" {
		t.Fatalf("Expected '%s', got '%s'", "string", b.StringPtr())
	}
}

func TestWriteByte(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteByte('s')
	b.WriteByte('t')
	b.WriteByte('r')
	b.WriteByte('i')
	b.WriteByte('n')
	b.WriteByte('g')
	if b.StringPtr() != "string" {
		t.Fatalf("Expected '%s', got '%s'", "string", b.StringPtr())
	}
}

func TestWriteRune(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteRune('s')
	b.WriteRune('t')
	b.WriteRune('r')
	b.WriteRune('i')
	b.WriteRune('n')
	b.WriteRune('g')
	b.WriteRune('👍')
	if b.StringPtr() != "string👍" {
		t.Fatalf("Expected '%s', got '%s'", "string", b.StringPtr())
	}
}

func TestWriteAt(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteAt([]byte("string"), 0)
	b.WriteAt([]byte("string"), 1)
	b.WriteAt([]byte("string"), 4)
	if b.StringPtr() != "sstrstring" {
		t.Fatalf("Expected '%s', got '%s'", "sstrstring", b.StringPtr())
	}
	b.Reset()

	b.WriteStringAt("string", 0)
	b.WriteStringAt("string", 1)
	b.WriteStringAt("string", 4)
	if b.StringPtr() != "sstrstring" {
		t.Fatalf("Expected '%s', got '%s'", "sstrstring", b.StringPtr())
	}
	b.Reset()
}

func TestTruncate(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteString("string")
	b.Truncate(2)
	if b.StringPtr() != "stri" {
		t.Fatalf("Expected '%s', got '%s'", "stri", b.StringPtr())
	}

	b.Truncate(0)
	if b.StringPtr() != "stri" {
		t.Fatalf("Expected '%s', got '%s'", "stri", b.StringPtr())
	}
}

func TestDeleteByte(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteString("string")
	b.DeleteByte(3)
	b.DeleteByte(0)
	if b.StringPtr() != "trng" {
		t.Fatalf("Expected '%s', got '%s'", "trng", b.StringPtr())
	}
}

func TestDelete(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteString("string")
	b.Delete(2, 2)
	if b.StringPtr() != "stng" {
		t.Fatalf("Expected '%s', got '%s'", "stng", b.StringPtr())
	}
}

func TestInsertByte(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteString("string")
	b.InsertByte(0, 's')
	b.InsertByte(3, '_')
	if b.StringPtr() != "sst_ring" {
		t.Fatalf("Expected '%s', got '%s'", "sst_ring", b.StringPtr())
	}
}

func TestInsert(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteString("string")
	b.Insert(0, []byte("__"))
	b.Insert(3, []byte("--"))
	if b.StringPtr() != "__s--tring" {
		t.Fatalf("Expected '%s', got '%s'", "__s--tring", b.StringPtr())
	}
}

func TestString(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	b.WriteString("string")
	if b.StringPtr() != "string" || b.String() != b.StringPtr() {
		t.Fatalf("Expected '%s', got '%s'", b.String(), b.StringPtr())
	}
}

func TestGuarantee(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0))
	b.Guarantee(10)
	if b.Len() != 0 || b.Cap() != 10 {
		t.Fatalf("Expected len:%d cap%d, got len:%d cap:%d", 0, 10, b.Len(), b.Cap())
	}
}

func TestGrow(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0))
	b.Grow(10)
	if b.Len() != 10 || b.Cap() != 10 {
		t.Fatalf("Expected len:%d cap%d, got len:%d cap:%d", 10, 10, b.Len(), b.Cap())
	}
}

func TestToLower(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))

	b.WriteString("STRING")
	bytes.ToLower(b.B)
	if b.StringPtr() != "string" {
		t.Fatalf("Expected 'string', got '%s'", b.StringPtr())
	}
	b.Reset()

	b.WriteString("STriNG!1")
	bytes.ToLower(b.B)
	if b.StringPtr() != "string!1" {
		t.Fatalf("Expected 'string!1', got '%s'", b.StringPtr())
	}
	b.Reset()

	b.WriteString("STRING!1234567890_+-=!£$%^&*()[]{}")
	bytes.ToLower(b.B)
	if b.StringPtr() != "string!1234567890_+-=!£$%^&*()[]{}" {
		t.Fatalf("Expected 'string!1234567890_+-=!£$%%^&*()[]{}', got '%s'", b.StringPtr())
	}
	b.Reset()
}

func TestToUppwer(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 0, 1024))

	b.WriteString("string")
	bytes.ToUpper(b.B)
	if b.StringPtr() != "STRING" {
		t.Fatalf("Expected 'STRING', got '%s'", b.StringPtr())
	}
	b.Reset()

	b.WriteString("STriNG!1")
	bytes.ToUpper(b.B)
	if b.StringPtr() != "STRING!1" {
		t.Fatalf("Expected 'STRING!1', got '%s'", b.StringPtr())
	}
	b.Reset()

	b.WriteString("string!1234567890_+-=!£$%^&*()[]{}")
	bytes.ToUpper(b.B)
	if b.StringPtr() != "STRING!1234567890_+-=!£$%^&*()[]{}" {
		t.Fatalf("Expected 'STRING!1234567890_+-=!£$%%^&*()[]{}', got '%s'", b.StringPtr())
	}
	b.Reset()
}
