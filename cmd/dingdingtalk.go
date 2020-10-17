package cmd

import (
	"fmt"
	dingtalk_robot "github.com/JetBlink/dingtalk-notify-go-sdk"
)

func SendToDingdingTalk(content string)  {
	msg := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
		"at": map[string]interface{}{
			"atMobiles": []string{},
			"isAtAll":   false,
		},
	}
	robot := dingtalk_robot.NewRobot("109dd44ba4ee796d034100added0e0ad4644102f854c9e9ae0be2688de0b639e", "SEC9a6e556a9d3ce71b5d1e9ff5704a91b29fd5d600afed4c930b39ea639f8ec398")
	if err := robot.SendMessage(msg); err != nil {
		fmt.Println(err)
	}
}