# Security Features

The RSS Aggregator includes comprehensive security protections designed for production environments with anonymous access.

## 🔒 Security Overview

The service implements multiple layers of security to protect against common web application vulnerabilities and abuse:

### **Rate Limiting**
- **Per-IP rate limiting** to prevent abuse and DoS attacks
- **Configurable limits**: Default 10 requests/second with 20 request burst
- **Automatic IP detection** from forwarded headers (X-Forwarded-For, X-Real-IP)
- **429 Too Many Requests** response when limits exceeded

### **Input Validation**
- **OData query parameter validation** with length limits
- **Path parameter sanitization** for topic names
- **Request size limits** (default 10MB) to prevent large payload attacks
- **Type validation** for numeric parameters ($top, $skip)

### **Security Headers**
- **XSS Protection**: Browser XSS filter enabled
- **Content Type Sniffing Prevention**: Prevents MIME type sniffing attacks
- **Frame Denial**: Prevents clickjacking attacks
- **Content Security Policy**: Restricts resource loading to same origin
- **Strict Transport Security**: HSTS headers for HTTPS enforcement
- **Referrer Policy**: Controls referrer information leakage

### **CORS Protection**
- **Configurable origins**: Default allows all origins (*)
- **Method restrictions**: Only GET, POST, OPTIONS allowed
- **Header restrictions**: Controlled set of allowed headers
- **Exposed headers**: Request ID for debugging

### **Request Tracking**
- **Unique request IDs** for monitoring and debugging
- **Security logging** with IP, method, path, status, latency
- **Error tracking** for failed requests
- **User agent logging** for client identification

## 🛡️ Protection Against Common Attacks

### **DDoS Protection**
```bash
# Rate limiting prevents rapid requests
RATE_LIMIT_PER_SECOND=10.0  # 10 requests per second
RATE_LIMIT_BURST=20         # Allow bursts up to 20 requests
```

### **Injection Attacks**
- **OData parameter validation** prevents query injection
- **Path parameter sanitization** prevents path traversal
- **Request size limits** prevent large payload attacks

### **Information Disclosure**
- **Security headers** prevent sensitive information leakage
- **Error messages** are generic to avoid information disclosure
- **Request IDs** help with debugging without exposing internals

### **Cross-Site Attacks**
- **CORS configuration** prevents unauthorized cross-origin requests
- **Content Security Policy** prevents XSS and resource injection
- **Frame denial** prevents clickjacking attacks

## ⚙️ Configuration

### **Environment Variables**

```bash
# Rate Limiting
ENABLE_RATE_LIMIT=true              # Enable/disable rate limiting
RATE_LIMIT_PER_SECOND=10.0          # Requests per second per IP
RATE_LIMIT_BURST=20                 # Burst limit per IP

# CORS Configuration
ENABLE_CORS=true                    # Enable/disable CORS
ALLOWED_ORIGINS=*                   # Comma-separated list of origins

# Security Headers
ENABLE_SECURITY_HEADERS=true        # Enable/disable security headers
MAX_REQUEST_SIZE=10485760           # Maximum request size (10MB)
ENABLE_REQUEST_ID=true              # Enable request ID tracking
```

### **Production Recommendations**

```bash
# Stricter rate limiting for production
RATE_LIMIT_PER_SECOND=5.0           # 5 requests per second
RATE_LIMIT_BURST=10                 # Smaller burst allowance

# Restrict CORS origins
ALLOWED_ORIGINS=https://yourdomain.com,https://api.yourdomain.com

# Enable HTTPS redirect in production
# Set SSLRedirect=true in security headers when using HTTPS
```

## 📊 Monitoring and Logging

### **Security Log Format**
```
ip=192.168.1.1 method=GET path=/api/v1/feeds/tech status=200 latency=15ms user_agent=Mozilla/5.0...
ip=192.168.1.2 method=GET path=/api/v1/feeds/invalid status=400 latency=5ms user_agent=curl/7.68.0 error=true
```

### **Key Metrics to Monitor**
- **Rate limit violations** (429 responses)
- **Invalid input attempts** (400 responses)
- **Large request attempts** (413 responses)
- **Request patterns** by IP address
- **Error rates** and response times

### **Request ID Tracking**
- **Unique identifier** for each request
- **Exposed in response headers** as X-Request-ID
- **Logged with all security events** for correlation
- **Helps with debugging** and incident response

## 🔧 Behind Proxy/Load Balancer

The service properly handles requests behind proxies and load balancers:

### **IP Address Detection**
1. **X-Forwarded-For** header (primary)
2. **X-Real-IP** header (fallback)
3. **X-Client-IP** header (fallback)
4. **Remote address** (final fallback)

### **Forwarded Headers**
```bash
# Example with multiple IPs in X-Forwarded-For
X-Forwarded-For: 192.168.1.1, 10.0.0.1, 172.16.0.1
# Service uses: 192.168.1.1 (first IP)
```

## 🚨 Incident Response

### **Rate Limit Violations**
- **Monitor 429 responses** for abuse patterns
- **Check IP addresses** for suspicious activity
- **Consider blocking** persistent offenders
- **Adjust limits** if legitimate traffic is affected

### **Invalid Input Attempts**
- **Monitor 400 responses** for attack patterns
- **Check request patterns** for automated attacks
- **Review logs** for injection attempts
- **Update validation** if new attack vectors discovered

### **Large Request Attempts**
- **Monitor 413 responses** for DoS attempts
- **Check request sources** for malicious clients
- **Consider reducing** MAX_REQUEST_SIZE if needed

## 🔍 Security Testing

### **Rate Limiting Test**
```bash
# Test rate limiting
for i in {1..25}; do
  curl -H "X-Forwarded-For: 192.168.1.1" http://localhost:8080/health
done
# Should see 429 responses after limit exceeded
```

### **Input Validation Test**
```bash
# Test invalid OData parameters
curl "http://localhost:8080/api/v1/feeds/tech?\$top=abc"
# Should return 400 Bad Request

# Test invalid topic name
curl "http://localhost:8080/api/v1/feeds/invalid@topic"
# Should return 400 Bad Request
```

### **Security Headers Test**
```bash
# Check security headers
curl -I http://localhost:8080/health
# Should include X-Frame-Options, X-Content-Type-Options, etc.
```

## 📋 Security Checklist

- [ ] **Rate limiting enabled** and configured appropriately
- [ ] **CORS configured** for your domain(s)
- [ ] **Security headers enabled** and configured
- [ ] **Request size limits** set appropriately
- [ ] **Input validation** working for all endpoints
- [ ] **Logging configured** for security monitoring
- [ ] **HTTPS enabled** in production (SSLRedirect=true)
- [ ] **Monitoring alerts** set up for security events
- [ ] **Incident response plan** documented
- [ ] **Regular security reviews** scheduled

## 🔐 Additional Security Considerations

### **WAF Integration**
While this service includes basic protections, consider using a Web Application Firewall (WAF) for additional security:

- **Cloudflare** for DDoS protection and additional filtering
- **AWS WAF** for AWS deployments
- **Azure Application Gateway** for Azure deployments
- **Custom WAF rules** for specific attack patterns

### **Network Security**
- **Firewall rules** to restrict access
- **VPN access** for administrative functions
- **Network segmentation** to isolate services
- **Intrusion detection** for network monitoring

### **Application Security**
- **Regular updates** of dependencies
- **Security scanning** of container images
- **Code review** for security issues
- **Penetration testing** of deployed services

This security implementation provides a solid foundation for production deployment while maintaining flexibility for different deployment scenarios. 