package pansoubridge

import (
	"sort"

	"github.com/gin-gonic/gin"
	"pansou/config"
	"pansou/plugin"
)

var supportedCloudTypes = []string{"baidu", "aliyun", "115", "123", "xunlei", "quark", "guangya", "mobile", "tianyi", "uc", "pikpak", "ed2k", "magnet", "other"}

func MetaHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		plugins := plugin.GetRegisteredPlugins()
		names := make([]string, 0, len(plugins))
		for _, p := range plugins {
			names = append(names, p.Name())
		}
		sort.Strings(names)
		pluginOptions := make([]gin.H, 0, len(names))
		for _, name := range names {
			pluginOptions = append(pluginOptions, gin.H{"name": name})
		}
		var channels, defaults []string
		if config.AppConfig != nil {
			channels = append([]string(nil), config.AppConfig.DefaultChannels...)
			defaults = append([]string(nil), config.AppConfig.EnabledPlugins...)
		}
		if defaults == nil {
			defaults = []string{}
		}
		if channels == nil {
			channels = []string{}
		}
		c.JSON(200, gin.H{"data": gin.H{"plugins": pluginOptions, "channels": channels, "cloudTypes": supportedCloudTypes, "defaults": gin.H{"plugins": defaults, "channels": channels, "cloudTypes": supportedCloudTypes}}})
	}
}
