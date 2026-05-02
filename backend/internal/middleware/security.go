package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"opsmanage/internal/config"
	"opsmanage/internal/model"
)

var (
	cachedRules []model.SecurityRule
	rulesMu     sync.RWMutex
)

// ReloadRules refreshes the in-memory security rule cache from DB.
func ReloadRules() {
	var rules []model.SecurityRule
	config.DB.Where("status = ?", "enabled").Order("priority desc, id desc").Find(&rules)
	rulesMu.Lock()
	cachedRules = rules
	rulesMu.Unlock()
	log.Printf("安全规则已刷新，加载 %d 条", len(rules))
}

// SecurityCheck is a Gin middleware that enforces enabled security rules.
func SecurityCheck() gin.HandlerFunc {
	// Load rules on first use
	ReloadRules()

	return func(c *gin.Context) {
		rulesMu.RLock()
		rules := cachedRules
		rulesMu.RUnlock()

		clientIP := c.ClientIP()
		urlPath := c.Request.URL.Path
		userAgent := c.GetHeader("User-Agent")

		// Collect whitelist and blacklist rules separately
		var ipWhitelist []string
		var ipBlacklist []string
		var urlBlacklist []string
		var uaBlacklist []string

		for _, r := range rules {
			switch r.Type {
			case "ip_whitelist":
				ipWhitelist = append(ipWhitelist, splitContent(r.Content)...)
			case "ip_blacklist":
				ipBlacklist = append(ipBlacklist, splitContent(r.Content)...)
			case "url_blacklist":
				urlBlacklist = append(urlBlacklist, splitContent(r.Content)...)
			case "ua_blacklist":
				uaBlacklist = append(uaBlacklist, splitContent(r.Content)...)
			}
		}

		// IP whitelist: if any whitelist exists, IP must match
		if len(ipWhitelist) > 0 && !matchIP(clientIP, ipWhitelist) {
			log.Printf("[安全拦截] IP %s 不在白名单中, URL: %s", clientIP, urlPath)
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "访问被拒绝"})
			c.Abort()
			return
		}

		// IP blacklist
		if matchIP(clientIP, ipBlacklist) {
			log.Printf("[安全拦截] IP %s 在黑名单中, URL: %s", clientIP, urlPath)
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "访问被拒绝"})
			c.Abort()
			return
		}

		// URL blacklist
		if matchPattern(urlPath, urlBlacklist) {
			log.Printf("[安全拦截] URL %s 命中黑名单, IP: %s", urlPath, clientIP)
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "访问被拒绝"})
			c.Abort()
			return
		}

		// UA blacklist
		if matchPattern(userAgent, uaBlacklist) {
			log.Printf("[安全拦截] UA 命中黑名单, IP: %s, URL: %s", clientIP, urlPath)
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "访问被拒绝"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// splitContent splits a rule's content by comma or newline into trimmed entries.
func splitContent(content string) []string {
	var result []string
	for _, s := range strings.FieldsFunc(content, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// matchIP checks if the client IP matches any entry (exact or CIDR).
func matchIP(clientIP string, entries []string) bool {
	ip := net.ParseIP(clientIP)
	for _, entry := range entries {
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err == nil && cidr.Contains(ip) {
				return true
			}
		} else if clientIP == entry {
			return true
		}
	}
	return false
}

// matchPattern checks if text contains any of the patterns (case-insensitive substring match).
func matchPattern(text string, patterns []string) bool {
	lower := strings.ToLower(text)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}
