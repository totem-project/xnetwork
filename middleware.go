package xnetwork

import (
	"encoding/hex"
	"fmt"
)

func responseLogger(resp *Response, c *Client) error {
	if c.Debug {
		req := resp.Request
		reqLog := "\n==============================================================================\n" +
			"--- REQUEST ---\n" +
			fmt.Sprintf("SourceAddr: %s\n", resp.SourceAddr().String()) +
			fmt.Sprintf("DestinationAddr: %s\n", resp.DestinationAddr().String()) +
			fmt.Sprintf("RequestSendAt: %s\n", req.sendAt) +
			fmt.Sprintf("RequestRaw:\n%s\n", req.GetRaw()) +
			fmt.Sprintf("RequestHexDump:\n%s\n", hex.Dump(req.GetRaw())) +
			"------------------------------------------------------------------------------\n" +
			"--- RESPONSE ---\n" +
			fmt.Sprintf("ResponseReceiveAt: %s\n", resp.ReadAt) +
			fmt.Sprintf("TIME DURATION: %s\n", resp.GetLatency()) +
			fmt.Sprintf("ResponseString:\n%s\n", resp.GetRaw()) +
			fmt.Sprintf("ResponseHexDump:\n%s\n", hex.Dump(resp.GetRaw())) +
			"------------------------------------------------------------------------------\n"
		fmt.Println(reqLog)
	}
	return nil
}
