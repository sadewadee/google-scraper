# Plan: Expand Email Validation Regex Patterns

## Objective
Expand regex patterns di `gmaps/emailjob.go` untuk mendeteksi lebih banyak domain placeholder, sentry, trapmail, dan file extensions yang tidak valid.

## File Target
- [gmaps/emailjob.go](gmaps/emailjob.go) (lines 148-176)

## Pattern Baru yang Akan Ditambahkan

### 1. Sentry/Error Tracking Domains
```go
regexp.MustCompile(`@.*sentry\.io$`),
regexp.MustCompile(`@.*sentry\..+$`),  // any sentry subdomain
regexp.MustCompile(`@.*bugsnag\.com$`),
regexp.MustCompile(`@.*rollbar\.com$`),
regexp.MustCompile(`@.*errortracking\..+$`),
```

### 2. Trapmail/Spam Trap Domains
```go
regexp.MustCompile(`@.*trapmail\..+$`),
regexp.MustCompile(`@.*spamtrap\..+$`),
regexp.MustCompile(`@.*honeypot\..+$`),
regexp.MustCompile(`@mailinator\.com$`),
regexp.MustCompile(`@guerrillamail\..+$`),
regexp.MustCompile(`@tempmail\..+$`),
regexp.MustCompile(`@throwaway\..+$`),
regexp.MustCompile(`@fakeinbox\.com$`),
regexp.MustCompile(`@temp-mail\..+$`),
regexp.MustCompile(`@10minutemail\.com$`),
regexp.MustCompile(`@yopmail\.com$`),
regexp.MustCompile(`@maildrop\.cc$`),
regexp.MustCompile(`@dispostable\.com$`),
```

### 3. Placeholder Domains Tambahan
```go
regexp.MustCompile(`@placeholder\..+$`),
regexp.MustCompile(`@dummy\..+$`),
regexp.MustCompile(`@fake\..+$`),
regexp.MustCompile(`@testing\..+$`),
regexp.MustCompile(`@localhost$`),
regexp.MustCompile(`@127\.0\.0\.1$`),
regexp.MustCompile(`@demo\..+$`),
regexp.MustCompile(`@sitename\..+$`),
regexp.MustCompile(`@mywebsite\..+$`),
regexp.MustCompile(`@changeme\..+$`),
regexp.MustCompile(`@youremail\..+$`),
regexp.MustCompile(`@your-email\..+$`),
regexp.MustCompile(`@mail\.com$`),  // generic
```

### 4. File Extensions (False Positives)
```go
// Existing: png, jpg, jpeg, gif, svg, webp
// Add:
regexp.MustCompile(`\.(ico|bmp|tiff|tif|eps|ai|psd|pdf|doc|docx)$`),
regexp.MustCompile(`\.(mp3|mp4|wav|avi|mov|wmv|flv|mkv)$`),
regexp.MustCompile(`\.(css|js|json|xml|html|htm|php|asp|aspx)$`),
regexp.MustCompile(`\.(zip|rar|7z|tar|gz)$`),
```

### 5. CMS/Platform/Hosting Generic Addresses
```go
// Website Builders
regexp.MustCompile(`@wix\.com$`),
regexp.MustCompile(`@squarespace\.com$`),
regexp.MustCompile(`@weebly\.com$`),
regexp.MustCompile(`@webflow\.com$`),
regexp.MustCompile(`@duda\.co$`),
regexp.MustCompile(`@jimdo\.com$`),
regexp.MustCompile(`@site123\.com$`),
regexp.MustCompile(`@strikingly\.com$`),
regexp.MustCompile(`@carrd\.co$`),
regexp.MustCompile(`@webnode\.com$`),
regexp.MustCompile(`@zyro\.com$`),
regexp.MustCompile(`@tilda\.cc$`),
regexp.MustCompile(`@ucraft\.com$`),

// E-commerce Platforms
regexp.MustCompile(`@shopify\.com$`),
regexp.MustCompile(`@bigcommerce\.com$`),
regexp.MustCompile(`@woocommerce\.com$`),
regexp.MustCompile(`@magento\.com$`),
regexp.MustCompile(`@prestashop\.com$`),
regexp.MustCompile(`@ecwid\.com$`),
regexp.MustCompile(`@volusion\.com$`),
regexp.MustCompile(`@3dcart\.com$`),
regexp.MustCompile(`@bigcartel\.com$`),
regexp.MustCompile(`@storenvy\.com$`),

// CMS Platforms
regexp.MustCompile(`@wordpress\.org$`),
regexp.MustCompile(`@drupal\.org$`),
regexp.MustCompile(`@joomla\.org$`),
regexp.MustCompile(`@ghost\.org$`),
regexp.MustCompile(`@contentful\.com$`),
regexp.MustCompile(`@strapi\.io$`),
regexp.MustCompile(`@sanity\.io$`),
regexp.MustCompile(`@prismic\.io$`),
regexp.MustCompile(`@netlify\.com$`),
regexp.MustCompile(`@vercel\.com$`),

// Hosting Providers
regexp.MustCompile(`@godaddy\.com$`),
regexp.MustCompile(`@hostinger\.com$`),
regexp.MustCompile(`@bluehost\.com$`),
regexp.MustCompile(`@hostgator\.com$`),
regexp.MustCompile(`@namecheap\.com$`),
regexp.MustCompile(`@siteground\.com$`),
regexp.MustCompile(`@dreamhost\.com$`),
regexp.MustCompile(`@ionos\.com$`),
regexp.MustCompile(`@a2hosting\.com$`),
regexp.MustCompile(`@inmotionhosting\.com$`),
regexp.MustCompile(`@hover\.com$`),
regexp.MustCompile(`@register\.com$`),
regexp.MustCompile(`@domain\.com$`),
regexp.MustCompile(`@networksolutions\.com$`),
regexp.MustCompile(`@cloudflare\.com$`),
regexp.MustCompile(`@digitalocean\.com$`),
regexp.MustCompile(`@linode\.com$`),
regexp.MustCompile(`@vultr\.com$`),
regexp.MustCompile(`@aws\.amazon\.com$`),
regexp.MustCompile(`@heroku\.com$`),

// Landing Page Builders
regexp.MustCompile(`@unbounce\.com$`),
regexp.MustCompile(`@leadpages\.com$`),
regexp.MustCompile(`@instapage\.com$`),
regexp.MustCompile(`@clickfunnels\.com$`),
regexp.MustCompile(`@landingi\.com$`),

// Form/Survey Builders
regexp.MustCompile(`@typeform\.com$`),
regexp.MustCompile(`@jotform\.com$`),
regexp.MustCompile(`@wufoo\.com$`),
regexp.MustCompile(`@formstack\.com$`),
regexp.MustCompile(`@cognito\.com$`),
regexp.MustCompile(`@surveymonkey\.com$`),
regexp.MustCompile(`@google\.com$`),  // forms, etc
```

### 6. Update String Contains Check (line 204-208)
Tambahkan substring check:
- `"trapmail"`
- `"spamtrap"`
- `"honeypot"`
- `"tempmail"`
- `"disposable"`

## Verification
1. Build project: `go build -o gmaps-scraper`
2. Run tests: `go test ./gmaps/...`

## Implementation Notes
- Semua pattern case-insensitive karena email sudah di-lowercase di `isValidBusinessEmail()`
- Pattern menggunakan `$` untuk match akhir string
- Pattern menggunakan `\.` untuk escape dot literal
