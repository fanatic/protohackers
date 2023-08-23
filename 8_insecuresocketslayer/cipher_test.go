package insecuresocketslayer

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCipherSpecs(t *testing.T) {
	t.Run("hello - xor(1)", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		crw := NewCipherReaderWriter(b, []byte{0x02, 0x01})
		require.NoError(t, crw.Validate())
		_, err := crw.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x69, 0x64, 0x6d, 0x6d, 0x6e}, b.Bytes())
	})

	t.Run("hello - xor(1),reversebits", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		crw := NewCipherReaderWriter(b, []byte{0x02, 0x01, 0x01})
		require.NoError(t, crw.Validate())
		_, err := crw.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x96, 0x26, 0xb6, 0xb6, 0x76}, b.Bytes())
	})

	t.Run("hello - addpos", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		crw := NewCipherReaderWriter(b, []byte{0x05})
		require.NoError(t, crw.Validate())
		_, err := crw.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x68, 0x66, 0x6e, 0x6f, 0x73}, b.Bytes())
	})

	t.Run("hello - addpos,addpos", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		crw := NewCipherReaderWriter(b, []byte{0x05, 0x05})
		require.NoError(t, crw.Validate())
		_, err := crw.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x68, 0x67, 0x70, 0x72, 0x77}, b.Bytes())
	})

	invalidCipherSpecs := [][]byte{
		{},                                   // empty cipher spec
		{0x02, 0x00},                         // xor(0)
		{0x02, 0xab, 0x02, 0xab},             // xor(X),xor(X) for any X
		{0x01, 0x01},                         // reversebits,reversebits
		{0x02, 0xa0, 0x02, 0x0b, 0x02, 0xab}, // xor(A),xor(B),xor(C), where A|B=C
	}
	for _, cipherSpec := range invalidCipherSpecs {
		t.Run("invalid cipher specs", func(t *testing.T) {
			crw := NewCipherReaderWriter(nil, cipherSpec)
			require.Error(t, crw.Validate())
		})
	}

	t.Run("xor(123),addpos,reversebits", func(t *testing.T) {
		b := bytes.NewBuffer(nil)
		crw := NewCipherReaderWriter(b, []byte{0x02, 0x7b, 0x05, 0x01})
		require.NoError(t, crw.Validate())
		_, err := crw.Write([]byte("4x dog,5x car\n"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0xf2, 0x20, 0xba, 0x44, 0x18, 0x84, 0xba, 0xaa, 0xd0, 0x26, 0x44, 0xa4, 0xa8, 0x7e}, b.Bytes())
		b.Reset()

		_, err = crw.Write([]byte("3x rat,2x cat\n"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x6a, 0x48, 0xd6, 0x58, 0x34, 0x44, 0xd6, 0x7a, 0x98, 0x4e, 0x0c, 0xcc, 0x94, 0x31}, b.Bytes())
		b.Reset()

		crw.outOffset = 0
		_, err = crw.Write([]byte("5x car\n"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0x72, 0x20, 0xba, 0xd8, 0x78, 0x70, 0xee}, b.Bytes())
		b.Reset()

		_, err = crw.Write([]byte("3x rat\n"))
		require.NoError(t, err)
		assert.Equal(t, []byte{0xf2, 0xd0, 0x26, 0xc8, 0xa4, 0xd8, 0x7e}, b.Bytes())
		b.Reset()
	})

	t.Run("xor(123),addpos,reversebits (decode)", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{0xf2, 0x20, 0xba, 0x44, 0x18, 0x84, 0xba, 0xaa, 0xd0, 0x26, 0x44, 0xa4, 0xa8, 0x7e, 0x6a, 0x48, 0xd6, 0x58, 0x34, 0x44, 0xd6, 0x7a, 0x98, 0x4e, 0x0c, 0xcc, 0x94, 0x31})
		crw := NewCipherReaderWriter(b, []byte{0x02, 0x7b, 0x05, 0x01})
		require.NoError(t, crw.Validate())
		out, err := io.ReadAll(crw)
		require.NoError(t, err)
		assert.Equal(t, "4x dog,5x car\n3x rat,2x cat\n", string(out))

		b = bytes.NewBuffer([]byte{0x72, 0x20, 0xba, 0xd8, 0x78, 0x70, 0xee, 0xf2, 0xd0, 0x26, 0xc8, 0xa4, 0xd8, 0x7e})
		crw = NewCipherReaderWriter(b, []byte{0x02, 0x7b, 0x05, 0x01})
		require.NoError(t, crw.Validate())
		out, err = io.ReadAll(crw)
		require.NoError(t, err)
		assert.Equal(t, "5x car\n3x rat\n", string(out))
	})

	t.Run("reproducer 0simple", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{0x30, 0x31, 0x79, 0x21, 0x75, 0x6e, 0x78, 0x21, 0x62, 0x60, 0x73, 0x0b})
		crw := NewCipherReaderWriter(b, []byte{0x02, 0x01})
		require.NoError(t, crw.Validate())
		out, err := io.ReadAll(crw)
		require.NoError(t, err)
		assert.Equal(t, "10x toy car\n", string(out))

	})
}
