
package gofast;

import (
	"container/list"
)

type record_stack struct {
	l * list.List;
	byteOffset uint;
}

func NewRecordStack () * record_stack{
	var ret * record_stack;
	ret = new (record_stack);
	ret.l = list.New();
	ret.byteOffset = 0;
	return ret;
}

func (this * record_stack) push (rec * fcgi_record) {
	this.l.PushBack(rec);
}

func (this * record_stack) popBytes(nBytes uint) bool{
	if nBytes == 0 {
		return true;
	}
	if (this.l.Len() == 0) {
		return false;
	}
	for {
		elem := this.l.Front();
		if elem == nil{
			return false;
		}
		record := elem.Value.(* fcgi_record);
		bytesInRecord := uint(len(record.Body)) - this.byteOffset
		if bytesInRecord > nBytes {
			this.byteOffset += nBytes;
			//fmt.Println("Pop: Shifting ptr by", nBytes);
			return true;
		}
		//fmt.Println("Pop: Removing record containing", bytesInRecord, "valid bytes");
		nBytes -= bytesInRecord;
		this.byteOffset = 0;
		this.l.Remove(elem);
	}
	return false;
}

func (this * record_stack) sliceForByteAt(inOffset uint, outSlice ** []byte, outOffset * uint) bool{
	var record * fcgi_record;
	if (this.l.Len() == 0) {
		return false;
	}
	inOffset += this.byteOffset;
	for elem := this.l.Front(); elem != nil; elem = elem.Next(){
		record = elem.Value.(* fcgi_record);
		if uint(len(record.Body)) > inOffset{
			*outOffset = inOffset;
			*outSlice = &record.Body;
			return true;
		}
		inOffset -= uint(len(record.Body));
	}
	return false;
}

func (this * record_stack) readFixedSizeString(size uint32, out ** string, outBytes * uint) bool {
	s := "";
	*out = &s;
	var slice * []byte;
	var offset uint;
	var sizeCollected uint;
	sizeCollected = 0;
	for this.sliceForByteAt(*outBytes + sizeCollected, &slice, &offset){
		bytesInSlice := uint(len(*slice)) - offset;
		bytesToCollect := uint(size) - sizeCollected;
		//fmt.Println("bytesInSlice", bytesInSlice, "bytesToCollect", bytesToCollect);
		if bytesInSlice >= bytesToCollect {
			**out += string((*slice)[offset: offset + bytesToCollect]);
			sizeCollected += bytesToCollect
			*outBytes += sizeCollected;
			return true;
		}
		**out += string((*slice)[offset:]);
		sizeCollected += bytesInSlice;
	}
	return false;
}

func (this * record_stack) readVariantUInt32(out * uint32, outBytes * uint) bool{
	var slice * []byte;
	var offset uint;
	var b0, b1, b2, b3 uint32;
	if this.sliceForByteAt(*outBytes, &slice, &offset) {
		b0 = uint32((*slice)[offset]);
		if (b0 >> 7 == 0) {
			*out = b0;
			*outBytes += 1;
			return true;
		}
		b0 = b0 << 24;
		if (uint(len(*slice)) > offset + 3) {
			b1 = uint32((*slice)[offset + 1]) << 16;
			b2 = uint32((*slice)[offset + 2]) << 8;
			b3 = uint32((*slice)[offset + 3]);
			*out = b0 | b1 | b2 | b3;
			*outBytes += 4;
			return true;
		} else {
			var slice1, slice2, slice3 * []byte;
			var offset1, offset2, offset3 uint;
			if this.sliceForByteAt(*outBytes + 1, &slice1, &offset1) && this.sliceForByteAt(*outBytes + 2, &slice2, &offset2) && this.sliceForByteAt(*outBytes + 3, &slice3, &offset3) {
				b1 = uint32((*slice1)[offset1]) << 16;
				b2 = uint32((*slice2)[offset2]) << 8;
				b3 = uint32((*slice3)[offset3]);
				*out = b0 | b1 | b2 | b3;
				*outBytes += 4;
				return true;
			}
		}
	}
	return false;
}

func (this * record_stack) parseKeyValueStrings(m map[string]string) {
	var bytesUsed uint;
	var keySize, valueSize uint32;
	var key, value * string;
	bytesUsed = 0;
	for this.readVariantUInt32(&keySize, &bytesUsed) && this.readVariantUInt32(&valueSize, &bytesUsed) && this.readFixedSizeString(keySize, &key, &bytesUsed) && this.readFixedSizeString(valueSize, &value, &bytesUsed) {
		m[*key] = *value
		this.popBytes(bytesUsed);
		bytesUsed = 0;
	}
}