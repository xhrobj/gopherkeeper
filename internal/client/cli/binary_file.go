package cli

import "github.com/xhrobj/gopherkeeper/internal/client/binaryfile"

func readBinaryFile(path string) (string, []byte, error) {
	return binaryfile.Read(path)
}

func writeBinaryFile(path string, data []byte) error {
	return binaryfile.Write(path, data)
}
