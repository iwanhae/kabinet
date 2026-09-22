package lifecycle

import (
	"testing"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/wal"
)

func TestSelectConvertBatchBoundsInput(t *testing.T) {
	segments := []wal.Segment{{Size: 10}, {Size: 15}, {Size: 20}}
	batch, size := selectConvertBatch(segments, 24)
	if len(batch) != 2 || size != 25 {
		t.Fatalf("expected two segments totaling 25 bytes, got %d totaling %d", len(batch), size)
	}

	many := make([]wal.Segment, maxSegmentsPerConvert+1)
	for i := range many {
		many[i].Size = 1
	}
	batch, size = selectConvertBatch(many, int64(len(many)+1))
	if len(batch) != maxSegmentsPerConvert || size != maxSegmentsPerConvert {
		t.Fatalf("expected segment cap %d, got %d totaling %d", maxSegmentsPerConvert, len(batch), size)
	}
}

func TestSelectMergeBatchBoundsInput(t *testing.T) {
	files := []catalog.File{{Size: 40}, {Size: 40}, {Size: 40}, {Size: 40}}
	batch, size := selectMergeBatch(files, 100, 96)
	if len(batch) != 3 || size != 120 {
		t.Fatalf("expected three files totaling 120 bytes, got %d totaling %d", len(batch), size)
	}

	batch, size = selectMergeBatch(files, 1_000, 2)
	if len(batch) != 2 || size != 80 {
		t.Fatalf("expected file cap 2, got %d totaling %d", len(batch), size)
	}
}
