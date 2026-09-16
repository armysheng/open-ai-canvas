package protocol

func parseXAIVideoPoll(c PollContext, payload map[string]any) (PollResult, error) {
	result, err := parseAsyncMediaPoll(c, payload, CapabilityVideo)
	if err != nil {
		return PollResult{}, err
	}
	// xAI returns a singular video object; URL resolution and authenticated
	// downloads remain the host's responsibility.
	if video := object(payload["video"]); video != nil {
		if value := firstString(video, "url"); value != "" {
			result.Result = &Result{Videos: []MediaReference{{URL: value, Kind: "video"}}}
		}
	}
	return result, nil
}
