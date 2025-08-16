This package provides a ReadWriteCloser that can be reset, in that you are able to switch the underlying io.ReadWriteCloser you are using, without disrupting the application using the ResReadWriteCloser provided in this package.

There are a couple exceptions to this rule. Since Read and Write calls can be blocking, if the ResReadWriteCloser is reset during such operations, it will return ErrRWCReset in place of any error, nil or otherwise, returned by the prior ReadWriteCloser. The original number of bytes read or written are not discarded.

See the [package documentation](https://pkg.go.dev/github.com/tech10/rwc) for full docs.
