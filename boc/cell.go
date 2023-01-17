package boc

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
)

const BOCSizeLimit = 65536
const CellBits = 1023

var ErrCellRefsOverflow = errors.New("too many refs")
var ErrNotEnoughRefs = errors.New("not enough refs")
var ErrNotSingleRoot = errors.New("should be one root cell")

type CellType uint8

const (
	OrdinaryCell CellType = iota
	PrunedBranchCell
	LibraryCell
	MerkleProofCell
	MerkleUpdateCell
)

type Cell struct {
	bits      BitString
	refs      [4]*Cell
	refCursor int
	cellType  CellType
	// TODO: add capacity checking
}

func NewCell() *Cell {
	return &Cell{
		bits:     NewBitString(CellBits),
		refs:     [4]*Cell{},
		cellType: OrdinaryCell,
	}
}

func NewCellExotic(cellType CellType) *Cell {
	return &Cell{
		bits:     NewBitString(CellBits),
		refs:     [4]*Cell{},
		cellType: cellType,
	}
}

func (c *Cell) RefsSize() int {
	var count int
	for i := range c.refs {
		if c.refs[i] != nil {
			count++
		}
	}
	return count
}

func (c *Cell) Refs() []*Cell {
	res := make([]*Cell, 0, 4)
	for _, ref := range c.refs {
		if ref != nil {
			res = append(res, ref)
		}
	}
	return res
}

func (c *Cell) IsExotic() bool {
	return c.cellType != OrdinaryCell
}

func (c *Cell) CellType() CellType {
	return c.cellType
}

func (c *Cell) BitSize() int {
	return c.bits.GetWriteCursor()
}

func (c *Cell) Hash() ([]byte, error) {
	return hashCell(c)
}

func (c *Cell) HashString() (string, error) {
	h, err := hashCell(c) //todo: check error
	return hex.EncodeToString(h), err
}

func (c *Cell) ToBoc() ([]byte, error) {
	return SerializeBoc(c, false, false, false, 0)
}

func (c *Cell) ToBocString() (string, error) {
	return c.ToBocStringCustom(false, false, false, 0)
}

func (c *Cell) ToBocBase64() (string, error) {
	return c.ToBocBase64Custom(false, false, false, 0)
}

func (c *Cell) ToBocCustom(idx bool, hasCrc32 bool, cacheBits bool, flags uint) ([]byte, error) {
	return SerializeBoc(c, idx, hasCrc32, cacheBits, flags)
}

func (c *Cell) ToBocStringCustom(idx bool, hasCrc32 bool, cacheBits bool, flags uint) (string, error) {
	boc, err := c.ToBocCustom(idx, hasCrc32, cacheBits, flags)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(boc), nil
}

func (c *Cell) ToBocBase64Custom(idx bool, hasCrc32 bool, cacheBits bool, flags uint) (string, error) {
	boc, err := c.ToBocCustom(idx, hasCrc32, cacheBits, flags)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(boc), nil
}

func (c *Cell) NewRef() (*Cell, error) {
	n := NewCell()
	return n, c.AddRef(n)
}

func (c *Cell) AddRef(c2 *Cell) error {
	for i := range c.refs {
		if c.refs[i] == nil {
			c.refs[i] = c2
			return nil
		}
	}
	return ErrCellRefsOverflow
}

func (c *Cell) NextRef() (*Cell, error) {
	ref := c.refs[c.refCursor]
	if ref != nil {
		c.refCursor++
		ref.ResetCounters()
		return ref, nil
	}
	return nil, ErrNotEnoughRefs
}

func (c *Cell) toStringImpl(ident string, iterationsLimit *int) string {
	s := ident + "x{" + c.bits.ToFiftHex() + "}\n"
	if *iterationsLimit == 0 {
		return s
	}
	*iterationsLimit -= 1
	for _, ref := range c.Refs() {
		s += ref.toStringImpl(ident+" ", iterationsLimit)
	}
	return s
}

func (c *Cell) ToString() string {
	iter := BOCSizeLimit
	return c.toStringImpl("", &iter)
}

func (c *Cell) ReadUint(bitLen int) (uint64, error) {
	return c.bits.ReadUint(bitLen)
}

func (c *Cell) PickUint(bitLen int) (uint64, error) {
	return c.bits.PickUint(bitLen)
}

func (c *Cell) WriteUint(val uint64, bitLen int) error {
	return c.bits.WriteUint(val, bitLen)
}

func (c *Cell) WriteInt(val int64, bitLen int) error {
	return c.bits.WriteInt(val, bitLen)
}

func (c *Cell) setTopUppedArray(arr []byte, fulfilledBytes bool) error {
	err := c.bits.SetTopUppedArray(arr, fulfilledBytes)
	c.bits.cap = 1023
	return err
}

func (c *Cell) getBuffer() []byte {
	return c.bits.Buffer()
}

func (c *Cell) Skip(n int) error {
	return c.bits.Skip(n)
}

func (c *Cell) WriteBit(val bool) error {
	return c.bits.WriteBit(val)
}

func (c *Cell) ReadBit() (bool, error) {
	return c.bits.ReadBit()
}

func (c *Cell) ReadBits(n int) (BitString, error) {
	return c.bits.ReadBits(n)
}

func (c *Cell) RawBitString() BitString {
	return c.bits
}

func (c *Cell) WriteUnary(n uint) error {
	return c.bits.WriteUnary(n)
}
func (c *Cell) ReadUnary() (uint, error) {
	return c.bits.ReadUnary()
}

func (c *Cell) ReadLimUint(n int) (uint, error) {
	return c.bits.ReadLimUint(n)
}

func (c *Cell) WriteLimUint(val, n int) error {
	return c.bits.WriteLimUint(val, n)
}

func (c *Cell) WriteBitString(s BitString) error {
	return c.bits.WriteBitString(s)
}

func (c *Cell) WriteBigInt(val *big.Int, bitLen int) error {
	return c.bits.WriteBigInt(val, bitLen)
}

func (c *Cell) WriteBigUint(val *big.Int, bitLen int) error {
	return c.bits.WriteBigUint(val, bitLen)
}

func (c *Cell) ReadInt(n int) (int64, error) {
	return c.bits.ReadInt(n)
}

func (c *Cell) ReadBytes(n int) ([]byte, error) {
	return c.bits.ReadBytes(n)
}

func (c *Cell) ReadBigInt(n int) (*big.Int, error) {
	return c.bits.ReadBigInt(n)
}

func (c *Cell) ReadBigUint(n int) (*big.Int, error) {
	return c.bits.ReadBigUint(n)
}

func (c *Cell) ReadRemainingBits() BitString {
	return c.bits.ReadRemainingBits()
}

func (c *Cell) WriteBytes(b []byte) error {
	return c.bits.WriteBytes(b)
}

func (c *Cell) ResetCounters() {
	c.bits.ResetCounter()
	c.refCursor = 0
}

func (c *Cell) BitsAvailableForRead() int {
	return c.bits.BitsAvailableForRead()
}

func (c *Cell) RefsAvailableForRead() int {
	return c.RefsSize() - c.refCursor
}

func (c *Cell) Sign(key ed25519.PrivateKey) (BitString, error) {
	hash, err := c.Hash()
	if err != nil {
		return BitString{}, err
	}
	bs := NewBitString(512)
	err = bs.WriteBytes(ed25519.Sign(key, hash[:]))
	return bs, err
}

func (c *Cell) BitsAvailableForWrite() int {
	return c.bits.BitsAvailableForWrite()
}

func NewCellWithBits(b BitString) *Cell {
	if b.len > CellBits {
		panic("bit string not fit to Cell")
	}
	b.Grow(CellBits - b.len)
	return &Cell{bits: b, cellType: OrdinaryCell}
}
