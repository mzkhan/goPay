// Package isofields defines the ISO-8583:1987 field specification
// for communication with Visa and Mastercard acquirer hosts.
package isofields

import (
	"github.com/moov-io/iso8583"
	"github.com/moov-io/iso8583/encoding"
	"github.com/moov-io/iso8583/field"
	"github.com/moov-io/iso8583/padding"
	"github.com/moov-io/iso8583/prefix"
)

// Spec returns an ISO-8583:1987 message spec compatible with
// Visa Base I and Mastercard MIP.
func Spec() *iso8583.MessageSpec {
	return &iso8583.MessageSpec{
		Name: "goPay ISO-8583:1987",
		Fields: map[int]field.Field{
			0: field.NewString(&field.Spec{
				Length:      4,
				Description: "Message Type Indicator",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			1: field.NewBitmap(&field.Spec{
				Length:      16,
				Description: "Bitmap",
				Enc:         encoding.BytesToASCIIHex,
				Pref:        prefix.Hex.Fixed,
			}),
			2: field.NewString(&field.Spec{
				Length:      19,
				Description: "Primary Account Number",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.LL,
			}),
			3: field.NewNumeric(&field.Spec{
				Length:      6,
				Description: "Processing Code",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
				Pad:         padding.Left('0'),
			}),
			4: field.NewString(&field.Spec{
				Length:      12,
				Description: "Amount, Transaction",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
				Pad:         padding.Left('0'),
			}),
			7: field.NewString(&field.Spec{
				Length:      10,
				Description: "Transmission Date and Time",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			11: field.NewString(&field.Spec{
				Length:      6,
				Description: "Systems Trace Audit Number",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
				Pad:         padding.Left('0'),
			}),
			12: field.NewString(&field.Spec{
				Length:      6,
				Description: "Time, Local Transaction",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			13: field.NewString(&field.Spec{
				Length:      4,
				Description: "Date, Local Transaction",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			14: field.NewString(&field.Spec{
				Length:      4,
				Description: "Date, Expiration",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			22: field.NewString(&field.Spec{
				Length:      3,
				Description: "Point of Service Entry Mode",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			25: field.NewString(&field.Spec{
				Length:      2,
				Description: "Point of Service Condition Code",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			32: field.NewString(&field.Spec{
				Length:      11,
				Description: "Acquiring Institution ID Code",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.LL,
			}),
			35: field.NewString(&field.Spec{
				Length:      37,
				Description: "Track 2 Data",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.LL,
			}),
			37: field.NewString(&field.Spec{
				Length:      12,
				Description: "Retrieval Reference Number",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			38: field.NewString(&field.Spec{
				Length:      6,
				Description: "Authorization ID Response",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			39: field.NewString(&field.Spec{
				Length:      2,
				Description: "Response Code",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			41: field.NewString(&field.Spec{
				Length:      8,
				Description: "Card Acceptor Terminal ID",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
				Pad:         padding.Right(' '),
			}),
			42: field.NewString(&field.Spec{
				Length:      15,
				Description: "Card Acceptor ID Code",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
				Pad:         padding.Right(' '),
			}),
			43: field.NewString(&field.Spec{
				Length:      40,
				Description: "Card Acceptor Name/Location",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
				Pad:         padding.Right(' '),
			}),
			49: field.NewString(&field.Spec{
				Length:      3,
				Description: "Currency Code, Transaction",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			52: field.NewBinary(&field.Spec{
				Length:      8,
				Description: "PIN Data",
				Enc:         encoding.Binary,
				Pref:        prefix.Binary.Fixed,
			}),
			55: field.NewString(&field.Spec{
				Length:      999,
				Description: "ICC/EMV Data",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.LLL,
			}),
			63: field.NewString(&field.Spec{
				Length:      999,
				Description: "Private Use (Network Data)",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.LLL,
			}),
			90: field.NewString(&field.Spec{
				Length:      42,
				Description: "Original Data Elements",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.Fixed,
			}),
			127: field.NewString(&field.Spec{
				Length:      999,
				Description: "Network Token/Cryptogram Data",
				Enc:         encoding.ASCII,
				Pref:        prefix.ASCII.LLL,
			}),
		},
	}
}
