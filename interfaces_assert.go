package textio

// Compile-time interface assertions

var _ TokenReader = (*Reader)(nil)
var _ TokenStreamer = (*Reader)(nil)
var _ TokenReaderStreamer = (*Reader)(nil)

// If ReaderCloser exists and embeds Reader:
var _ TokenReader = (*ReaderCloser)(nil)
var _ TokenStreamer = (*ReaderCloser)(nil)
var _ TokenReaderStreamer = (*ReaderCloser)(nil)
var _ TokenReaderCloser = (*ReaderCloser)(nil)
var _ TokenStreamerCloser = (*ReaderCloser)(nil)
var _ TokenReaderStreamerCloser = (*ReaderCloser)(nil)
