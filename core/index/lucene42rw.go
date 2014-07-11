package index

import (
	"github.com/balzaczyy/golucene/core/codec/lucene40"
	"github.com/balzaczyy/golucene/core/codec/perfield"
	. "github.com/balzaczyy/golucene/core/codec/spi"
	. "github.com/balzaczyy/golucene/core/index/model"
)

// lucene42/Lucene42RWCodec.java

var dv = newLucene42RWDocValuesFormat()

// Read-write version of Lucene42Codec for testing.
var Lucene42RWCodec = &CodecImpl{
	fieldsFormat:     newLucene41StoredFieldsFormat(),
	vectorsFormat:    newLucene42TermVectorsFormat(),
	fieldInfosFormat: newLucene42FieldInfosFormat(),
	infosFormat:      lucene40.NewLucene40SegmentInfoFormat(),
	// liveDocsFormat: newLucene40LiveDocsFormat(),
	// Returns the postings format that should be used for writing new
	// segments of field.
	//
	// The default implemnetation always returns "Lucene41"
	postingsFormat: perfield.NewPerFieldPostingsFormat(func(field string) PostingsFormat {
		panic("not implemented yet")
	}),
	// Returns the decvalues format that should be used for writing new
	// segments of field.
	//
	// The default implementation always returns "Lucene42"
	docValuesFormat: perfield.NewPerFieldDocValuesFormat(func(field string) DocValuesFormat {
		return dv
	}),
	normsFormat: newLucene42NormsFormat(),
}

// lucene42/Lucene42RWDocValuesFormat.java

// Read-write version of Lucene42DocValuesFormat for testing
type Lucene42RWDocValuesFormat struct {
	*Lucene42DocValuesFormat
}

func (f *Lucene42RWDocValuesFormat) FieldsConsumer(state *SegmentWriteState) (w DocValuesConsumer, err error) {
	panic("not implemented yet")
}

func newLucene42RWDocValuesFormat() *Lucene42RWDocValuesFormat {
	return &Lucene42RWDocValuesFormat{
		NewLucene42DocValuesFormat(),
	}
}
