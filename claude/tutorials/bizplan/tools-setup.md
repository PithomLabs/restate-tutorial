# Tools Setup Guide

> **Complete step-by-step setup for all business tools**

---

## 🎯 Overview

This guide walks you through setting up every tool needed to run your tutorial business. Total setup time: ~4 hours.

**Total Cost:** $0/month (using free tiers)

---

## 📋 Master Checklist

- [ ] GitHub Organization
- [ ] Gumroad Store
- [ ] ConvertKit Account
- [ ] Discord Server
- [ ] GitHub Pages Website
- [ ] Analytics
- [ ] Support System
- [ ] Automation

---

## 1️⃣ GitHub Setup (30 mins)

### Create Organization

**Why:** Professional presence, separate from personal account

**Steps:**
1. Go to https://github.com/organizations/plan
2. Choose "Create a free organization"
3. Name: `restate-go-tutorials` (or similar)
4. Email: your business email
5. Choose "My personal account"

### Create Repositories

**Main Tutorial Repo:**
```bash
# Create repo via GitHub UI or CLI
gh repo create restate-go-tutorials/complete-tutorial --public

# Clone and push your content
git clone https://github.com/restate-go-tutorials/complete-tutorial
cd complete-tutorial
cp -r ~/path/to/your/tutorials/* .
git add .
git commit -m "Initial commit: Complete tutorial series"
git push
```

**Additional Repos:**
```bash
# Public repo for free modules (1-4)
gh repo create restate-go-tutorials/free-modules --public

# Private repo for paid modules (5-12) - paid customers get access
gh repo create restate-go-tutorials/pro-modules --private

# Solutions repository
gh repo create restate-go-tutorials/solutions --private

# Website/landing page
gh repo create restate-go-tutorials/website --public
```

### Configure Repository Settings

**For main tutorial repo:**

1. **About section:**
   - Description: "Complete Restate Go Tutorial Series - Master distributed systems"
   - Website: Your landing page URL
   - Topics: `restate`, `go`, `golang`, `distributed-systems`, `tutorial`

2. **Enable Features:**
   - ✅ Issues (for questions/bugs)
   - ✅ Discussions (for community forum)
   - ✅ Projects (for roadmap)
   - ❌ Wiki (not needed)

3. **Create Issue Templates:**
```bash
mkdir -p .github/ISSUE_TEMPLATE
```

Create `.github/ISSUE_TEMPLATE/bug_report.yml`:
```yaml
name: Bug Report
description: Report a problem with the tutorial code
labels: ["bug"]
body:
  - type: input
    id: module
    attributes:
      label: Module
      description: Which module is affected?
      placeholder: "e.g., Module 05: Sagas"
    validations:
      required: true
  
  - type: textarea
    id: description
    attributes:
      label: Description
      description: What went wrong?
    validations:
      required: true
  
  - type: textarea
    id: steps
    attributes:
      label: Steps to Reproduce
      description: How can we reproduce this?
    validations:
      required: true
```

Create `.github/ISSUE_TEMPLATE/question.yml`:
```yaml
name: Question
description: Ask a question about the tutorials
labels: ["question"]
body:
  - type: textarea
    id: question
    attributes:
      label: Your Question
    validations:
      required: true
```

4. **Enable GitHub Discussions:**
   - Settings → Features → Enable Discussions
   - Create categories: Questions, Show & Tell, Ideas, General

5. **Create Project Board:**
   - Projects → New Project → "Roadmap"
   - Add columns: Planned, In Progress, Done
   - Make it public so customers can see what's coming

### Create Professional README

Create `README.md` in main repo:

```markdown
# 🚀 Complete Restate Go Tutorial Series

Master distributed systems and durable execution with Restate in Go.

## 📚 What's Included

- 12 comprehensive modules (beginner → production)
- Production-ready code examples
- Hands-on exercises with solutions
- Video walkthroughs
- Active community support

## 🎯 Who Is This For?

Backend engineers with 1-5 years Go experience who want to:
- Build resilient distributed systems
- Learn durable execution patterns
- Master Restate framework
- Ship production-grade applications

## 🆓 Free Modules

Try before you buy! Modules 1-4 are completely free:

1. **Introduction to Restate** - Core concepts
2. **Services and Handlers** - Building services
3. **State Management** - Virtual objects
4. **Workflows** - Long-running processes

[Start learning →](./01-introduction)

## 💎 Pro Access

Get the complete series for $99 (one-time payment):

- ✅ All 12 modules
- ✅ Lifetime updates
- ✅ Private Discord community
- ✅ Priority support
- ✅ Solutions repository

[Get Pro Access →](https://your-gumroad.com/restate-go)

## 📖 Module Overview

| Module | Topic | Status |
|--------|-------|--------|
| 01 | Introduction to Restate | ✅ Free |
| 02 | Services and Handlers | ✅ Free |
| 03 | State Management | ✅ Free |
| 04 | Workflows | ✅ Free |
| 05 | Journaling | 🔒 Pro |
| 06 | Sagas & Compensation | 🔒 Pro |
| 07 | Idempotency | 🔒 Pro |
| 08 | External Integration | 🔒 Pro |
| 09 | Microservices Orchestration | 🔒 Pro |
| 10 | Observability | 🔒 Pro |
| 11 | Security | 🔒 Pro |
| 12 | Production & Deployment | 🔒 Pro |

## 🤝 Community

- [GitHub Discussions](../../discussions) - Ask questions
- [Discord](https://discord.gg/your-server) - Live chat
- [Twitter](https://twitter.com/your-handle) - Updates & tips

## 📝 License

Code examples: MIT License  
Tutorial content: All rights reserved

## ⭐ Support

If you find this helpful, please star the repository!

[![Star this repo](https://img.shields.io/github/stars/restate-go-tutorials/complete-tutorial?style=social)](https://github.com/restate-go-tutorials/complete-tutorial)
```

---

## 2️⃣ Gumroad Setup (20 mins)

### Create Account

1. Go to https://gumroad.com
2. Sign up with your business email
3. Complete profile:
   - Name: Your name or business name
   - Bio: Brief description
   - Profile picture
   - Cover image

### Create Product

1. Click "Create Product" → "Digital Product"

**Product Details:**
```
Name: Complete Restate Go Tutorial Series

Tagline: Master distributed systems with production-ready Restate tutorials

Description:
Build resilient, production-grade distributed applications with Restate in Go.

🎯 What You'll Get:
✅ 12 comprehensive modules (beginner → production)
✅ 50+ hours of content
✅ Production-ready code examples
✅ Hands-on exercises with solutions
✅ Private GitHub repository access
✅ Discord community support
✅ Lifetime updates (new modules added regularly)

📚 Module Breakdown:
[List all 12 modules with descriptions]

💡 Perfect For:
- Backend engineers with Go experience
- Developers building distributed systems
- Teams adopting Restate
- Anyone learning durable execution

🎓 Testimonials:
[Add once you have them]

Price: $99
Add to Cart Text: Get Instant Access
```

2. **Upload Cover Image:**
   - Create in Canva (free): 1600x900px
   - Show code snippet + Restate logo
   - Professional design

3. **Configure Settings:**
   - ✅ Collect email addresses
   - ✅ Send receipt automatically
   - ✅ Ask for testimonial after purchase
   - Product limit: Unlimited
   - License key: Enable (needed for GitHub access)

4. **Set up License Key Integration:**
   - Enable "License Keys"
   - We'll use this to grant GitHub repo access

### Create Tiers (Optional)

**Create second product: "Team License"**
- Name: Restate Go Tutorials - Team License
- Price: $499
- Includes: 5 license keys
- Everything in Pro

### Configure Purchase Flow

**After purchase, customers receive:**
1. Gumroad receipt with product files
2. License key (for GitHub access automation)
3. Welcome email (we'll set this up in ConvertKit)

**Product delivery:**
- Option 1: Upload ZIP of paid modules
- Option 2: Include PDF with GitHub access instructions
- Option 3: Automatically grant GitHub access (requires integration)

### Set Up Gumroad Email

**Purchase Email Template:**
```
Subject: 🎉 Welcome to Restate Go Tutorials!

Hi there!

Thanks for purchasing the Complete Restate Go Tutorial Series!

🚀 GET STARTED:

1. Access Your Content:
   - GitHub Repository: [WILL SEND VIA EMAIL]
   - Your License Key: {license_key}

2. Join the Community:
   - Discord: https://discord.gg/your-server
   - Use code: {license_key}

3. Start Learning:
   - Begin with Module 05
   - Check out the README for overview
   - Join Discord if you get stuck

📧 IMPORTANT: Check your email for GitHub access instructions (arriving in next 5 minutes).

Need help? Reply to this email or ping me in Discord!

Happy coding! 🚀

[Your Name]
```

### Set Up Affiliate Program (Optional)

- Enable affiliate program: 20% commission
- Students can refer friends
- Automated payouts via Gumroad

---

## 3️⃣ ConvertKit Setup (45 mins)

### Create Account

1. Go to https://convertkit.com
2. Sign up (free up to 1,000 subscribers)
3. Complete onboarding

### Create Forms

**Landing Page Signup Form:**

1. Create Form → "Inline Form"
2. Name: "Free Modules Signup"
3. Customize:
   ```
   Headline: Start Learning Restate for Free
   
   Subheadline: Get instant access to 4 beginner modules + exclusive tips
   
   Button: Get Free Access
   ```
4. Settings:
   - Success message: "Check your email! Module 1 is waiting for you."
   - Double opt-in: No (for faster access)

5. Get embed code for website

**Exit Intent Popup (Optional):**
- Trigger: User about to leave
- Offer: Free module + 10% off Pro

### Create Email Sequences

**Sequence 1: Free Module Welcome**

Create automated sequence for free signups:

**Email 1 (Immediate):**
```
Subject: 🚀 Your Restate Tutorial is Ready!

Hi!

Welcome to the Restate Go community! 

📚 Here's your free access:

Module 01: Introduction to Restate
→ https://github.com/restate-go-tutorials/free-modules/01-introduction

Module 02: Services and Handlers
→ https://github.com/restate-go-tutorials/free-modules/02-services

Module 03: State Management
→ https://github.com/restate-go-tutorials/free-modules/03-state

Module 04: Workflows
→ https://github.com/restate-go-tutorials/free-modules/04-workflows

🎯 START HERE: Begin with Module 01 and work your way through.

⭐ Quick tip: Star the repo so you can find it easily!

Have questions? Hit reply anytime!

Happy learning,
[Your Name]

P.S. Want the complete series? Pro members get 8 more modules + private Discord. [Check it out →]
```

**Email 2 (3 days later):**
```
Subject: How's Module 01 going?

Hey!

Just checking in - have you started Module 01 yet?

If you're stuck anywhere, don't hesitate to:
- Reply to this email
- Open a GitHub issue
- Ask in our Discord

💡 Pro Tip: The best way to learn is by typing out the code yourself (not just copy/paste).

Cheers,
[Your Name]
```

**Email 3 (1 week later):**
```
Subject: Ready for advanced topics?

Hi!

If you've completed the free modules, you're ready for the good stuff!

The Pro modules cover:
✅ Sagas & distributed transactions
✅ External API integration
✅ Production deployment
✅ Security best practices
...and more

[Get Pro Access ($99) →]

Not ready yet? No problem! Take your time.

[Your Name]
```

**Sequence 2: Pro Customer Onboarding**

**Email 1 (Immediate - triggered by Gumroad webhook):**
```
Subject: 🎉 Welcome to Restate Go Pro!

Hi!

Your GitHub access is ready!

🔓 ACCESS YOUR CONTENT:

1. Go to: https://github.com/restate-go-tutorials/pro-modules
2. Log in to GitHub
3. You should now have access!

If you don't have access yet, reply with your GitHub username and I'll add you manually.

💬 JOIN DISCORD:

Our private community is here: https://discord.gg/your-server

Use verification code: {license_key}

📚 WHERE TO START:

Begin with Module 05 (Journaling) and work sequentially.

Each module builds on previous concepts.

Need help? I'm here!

[Your Name]

```

**Email 2 (1 week later):**
```
Subject: Pro tip: Journaling patterns

Hey!

How's it going with the Pro modules?

Quick tip for Module 05 (Journaling):

The key insight is that restate.Run() makes non-deterministic code deterministic by journaling the result...

[Include helpful tip]

Stuck somewhere? Reply to this email!

[Your Name]
```

### Set Up Broadcast System

**Create "Newsletter" Broadcast:**
- Send weekly tips every Wednesday
- Segment: All subscribers
- Template: Helpful code snippet + announcement

### Create Tags

- `free-user` - Has free access only
- `pro-customer` - Purchased Pro
- `team-customer` - Purchased Team
- `engaged` - Opens emails regularly
- `at-risk` - Hasn't opened in 30 days

### Set Up Automation Rules

**Rule 1:** When someone purchases (via Gumroad webhook):
- Add tag: `pro-customer`
- Remove tag: `free-user`
- Trigger Pro welcome sequence

**Rule 2:** If email bounce:
- Remove from list
- Add to "Bounced" segment

**Rule 3:** Re-engagement campaign:
- If tagged `at-risk`
- Send: "Miss you! Here's what's new..."

---

## 4️⃣ Discord Setup (30 mins)

### Create Server

1. Open Discord → "+" → "Create My Own" → "For a club or community"
2. Server Name: "Restate Go Tutorials"
3. Upload icon: Restate logo or custom

### Create Channel Structure

```
📢 WELCOME & INFO
├─ 👋 welcome
├─ 📜 rules
├─ 📚 resources
├─ 🎯 start-here

💬 COMMUNITY
├─ 💬 general-chat
├─ 🙋 questions
├─ 💡 show-and-tell
├─ 🐛 bugs

📚 MODULES (Pro Only)
├─ 📖 module-05-journaling
├─ 📖 module-06-sagas
├─ 📖 module-07-idempotency
├─ ... (one for each Pro module)

🛠️ RESOURCES (Pro Only)
├─ 💎 pro-announcements
├─ 🎁 exclusive-content
├─ 📞 office-hours

🤖 ADMIN
├─ 📊 mod-chat
└─ 🔔 logs
```

### Configure Roles

**@everyone role:**
- Can view: Welcome, Rules, Resources
- Free tier access

**@Pro role:**
- Everything @everyone has
- Access to Pro channels
- Priority support

**@Team role:**
- Everything @Pro has
- Team-specific channel
- 1-on-1 support booking

**@Moderator role:**
- Manage messages
- Kick/ban users
- Access mod chat

### Set Up Welcome Message

Install MEE6 bot (free tier):
1. Add MEE6 to server
2. Set up welcome message in `#welcome`:
   ```
   Welcome {{user}}! 👋

   🎯 START HERE:
   1. Read <#rules>
   2. Check out <#resources>
   3. Introduce yourself in <#general-chat>
   
   💎 Pro Members: React with 🔑 to verify your license
   ```

### Set Up Verification

**For Pro members:**

Option 1: Manual verification
- They DM you their license key
- You assign @Pro role

Option 2: Automated (requires bot development)
- Custom bot checks license key against Gumroad API
- Auto-assigns role

**Quick manual setup:**
1. Create `#verify` channel
2. Pin message: "Pro members: DM me your Gumroad license key for Pro role"
3. Verify and assign manually (until automated)

### Create Server Rules

`#rules` channel:
```markdown
# 📜 Server Rules

1. **Be Respectful** - Treat everyone with kindness
2. **Stay On Topic** - Keep discussions related to Restate/Go
3. **No Spam** - Don't spam or self-promote
4. **Use Right Channels** - Post in appropriate channels
5. **Search First** - Check if your question was already answered
6. **English Only** - Keep conversations in English

Breaking rules = warning → temporary mute → ban

Questions? DM @Moderator
```

### Create Resource Links

`#resources` channel:
```markdown
# 📚 Resources

**Official Restate:**
- Documentation: https://docs.restate.dev
- GitHub: https://github.com/restatedev
- Discord: https://discord.gg/restate

**Tutorial Series:**
- Free Modules: https://github.com/restate-go-tutorials/free-modules
- Buy Pro: https://gumroad.com/your-product

**Helpful Links:**
- Go Documentation: https://go.dev/doc
- Distributed Systems: [Resource list]

**Weekly Events:**
- Office Hours: Every Friday 5 PM UTC (#office-hours)
- Q&A Session: First Monday of month
```

---

## 5️⃣ GitHub Pages Setup (1 hour)

### Create Website Repository

```bash
gh repo create restate-go-tutorials/restate-go-tutorials.github.io --public
cd restate-go-tutorials.github.io
```

### Build Simple Landing Page

**index.html:**
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Restate Go Tutorials - Master Distributed Systems</title>
    <meta name="description" content="Complete Restate Go tutorial series - Build production-ready distributed systems">
    
    <!-- Open Graph for social sharing -->
    <meta property="og:title" content="Restate Go Tutorials">
    <meta property="og:description" content="Master distributed systems with production-ready Restate tutorials">
    <meta property="og:image" content="https://your-site.com/og-image.png">
    
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            line-height: 1.6;
            color: #333;
        }
        
        .hero {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 100px 20px;
            text-align: center;
        }
        
        .hero h1 {
            font-size: 3em;
            margin-bottom: 20px;
        }
        
        .hero p {
            font-size: 1.3em;
            margin-bottom: 30px;
            opacity: 0.9;
        }
        
        .cta-button {
            display: inline-block;
            background: white;
            color: #667eea;
            padding: 15px 40px;
            text-decoration: none;
            border-radius: 5px;
            font-weight: bold;
            font-size: 1.1em;
            margin: 10px;
        }
        
        .cta-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(0,0,0,0.2);
        }
        
        .cta-button.secondary {
            background: transparent;
            border: 2px solid white;
            color: white;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 60px 20px;
        }
        
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 30px;
            margin-top: 40px;
        }
        
        .feature {
            text-align: center;
            padding: 30px;
        }
        
        .feature-icon {
            font-size: 3em;
            margin-bottom: 15px;
        }
        
        .modules {
            background: #f8f9fa;
        }
        
        .pricing {
            text-align: center;
        }
        
        .price-card {
            background: white;
            border-radius: 10px;
            padding: 40px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
            margin: 20px auto;
            max-width: 500px;
        }
        
        .price {
            font-size: 3em;
            font-weight: bold;
            color: #667eea;
        }
    </style>
</head>
<body>
    <div class="hero">
        <h1>🚀 Master Restate in Go</h1>
        <p>Build production-ready distributed systems with confidence</p>
        <a href="#free" class="cta-button">Start Free</a>
        <a href="https://gumroad.com/your-product" class="cta-button secondary">Get Pro ($99)</a>
    </div>
    
    <div class="container features">
        <div class="feature">
            <div class="feature-icon">📚</div>
            <h3>12 Comprehensive Modules</h3>
            <p>From basics to production deployment</p>
        </div>
        <div class="feature">
            <div class="feature-icon">💻</div>
            <h3>Production-Ready Code</h3>
            <p>Real-world examples, not toy demos</p>
        </div>
        <div class="feature">
            <div class="feature-icon">🎓</div>
            <h3>Hands-On Learning</h3>
            <p>Exercises, solutions, validations</p>
        </div>
        <div class="feature">
            <div class="feature-icon">💬</div>
            <h3>Active Community</h3>
            <p>Discord support & discussions</p>
        </div>
    </div>
    
    <div class="container modules">
        <h2>What You'll Learn</h2>
        <div class="features" style="margin-top: 30px;">
            <div>✅ Services & Handlers</div>
            <div>✅ State Management</div>
            <div>✅ Workflows</div>
            <div>✅ Journaling</div>
            <div>✅ Sagas & Compensation</div>
            <div>✅ Idempotency</div>
            <div>✅ External Integration</div>
            <div>✅ Orchestration</div>
            <div>✅ Observability</div>
            <div>✅ Security</div>
            <div>✅ Production Deployment</div>
            <div>✅ Best Practices</div>
        </div>
    </div>
    
    <div class="container pricing">
        <h2>Simple Pricing</h2>
        <div class="price-card">
            <h3>Complete Series</h3>
            <div class="price">$99</div>
            <p>One-time payment • Lifetime access</p>
            <ul style="text-align: left; margin: 30px 0;">
                <li>✅ All 12 modules</li>
                <li>✅ Production-ready code</li>
                <li>✅ Private Discord</li>
                <li>✅ Lifetime updates</li>
                <li>✅ Solutions repository</li>
            </ul>
            <a href="https://gumroad.com/your-product" class="cta-button">Get Instant Access</a>
        </div>
    </div>
    
    <!-- ConvertKit Form -->
    <div class="container" id="free" style="text-align: center; background: #f8f9fa; border-radius: 10px; padding: 60px 20px;">
        <h2>Try 4 Modules Free</h2>
        <p>Get instant access to beginner modules - no credit card required</p>
        <!-- Paste ConvertKit form embed code here -->
        <div id="convertkit-form"></div>
    </div>
    
    <script>
        // Simple analytics (privacy-friendly)
        // Replace with Plausible or GA code
    </script>
</body>
</html>
```

### Deploy to GitHub Pages

```bash
git add .
git commit -m "Initial landing page"
git push

# Enable GitHub Pages
# Settings → Pages → Source: main branch
```

Site will be live at: `https://restate-go-tutorials.github.io`

### Custom Domain (Optional)

1. Buy domain: `restate-go-tutorials.com` (~$12/year)
2. Add CNAME file: `echo "restate-go-tutorials.com" > CNAME`
3. Update DNS: Point to GitHub Pages
4. Enable HTTPS in GitHub settings

---

## 6️⃣ Analytics Setup (15 mins)

### Option 1: Plausible (Recommended - Privacy-friendly)

**Self-hosted (Free):**
```bash
# Deploy to free tier hosting (Railway, Fly.io)
git clone https://github.com/plausible/analytics
# Follow their docs
```

**Or use Plausible Cloud:**
- $9/mo for up to 10k monthly pageviews
- Privacy-friendly (no cookies)
- GDPR compliant

**Add to site:**
```html
<script defer data-domain="restate-go-tutorials.com" src="https://plausible.io/js/script.js"></script>
```

### Option 2: Google Analytics (Free)

1. Create GA4 property
2. Add tracking code to website
3. Set up conversions:
   - Email signup
   - Gumroad button click
   - GitHub repo visit

---

## 7️⃣ Support System Setup (20 mins)

### Enable GitHub Support

**Create Support Template:**

`.github/SUPPORT.md`:
```markdown
# Getting Help

## 🙋 Questions

Have a question? Here's how to get help:

1. **Check the FAQ**: [Link to FAQ]
2. **Search Issues**: Someone may have asked already
3. **Ask in Discord**: Fastest response
4. **Open an Issue**: For bugs or detailed questions

## 💬 Discord

Join our community: https://discord.gg/your-server

- General questions: #questions
- Show your work: #show-and-tell
- Bug reports: #bugs

## 📧 Email Support

For private matters: support@restate-go-tutorials.com

Response time:
- Pro members: 24 hours
- Free tier: Best effort

## 📚 Resources

- Documentation: https://docs.restate.dev
- Tutorial Repo: https://github.com/restate-go-tutorials
```

### Set Up Support Email

**Option 1: Gmail Alias (Free)**
- Create: restategohelp@gmail.com
- Forward to your main email

**Option 2: Custom Domain Email**
- Use Zoho Mail free tier
- support@yourdomain.com

### Create FAQ Document

Create `docs/FAQ.md` in repo:
```markdown
# Frequently Asked Questions

## Purchase & Access

**Q: How do I access the Pro modules after purchase?**
A: You'll receive a GitHub invite within 24 hours...

**Q: Can I get a refund?**
A: Yes, 30-day money-back guarantee...

## Technical

**Q: My code doesn't work. What should I do?**
A: First, check that...

[Add 20-30 common questions]
```

---

## 8️⃣ Automation Setup (30 mins)

### Gumroad → ConvertKit Integration

**Using Zapier (Free tier: 100 tasks/month):**

1. Create Zap: Gumroad Sale → ConvertKit
2. Trigger: New Sale in Gumroad
3. Action: Subscribe to ConvertKit
4. Add tag: `pro-customer`
5. Trigger: Pro welcome sequence

### Gumroad → GitHub Integration

**Manual for now (automate later with webhook + script):**

When sale occurs:
1. Gumroad sends you email
2. You manually invite customer to private repo
3. Or use license key system

**Future automation:**
- Webhook receives Gumroad sale
- Script invites GitHub user to repo
- Confirmation email sent

### Auto-Backup System

**Backup GitHub to another location weekly:**

```bash
# Create backup script
git clone --mirror https://github.com/restate-go-tutorials/complete-tutorial
tar -czf backup-$(date +%Y%m%d).tar.gz complete-tutorial.git
# Upload to cloud storage
```

---

## ✅ Final Checklist

### Pre-Launch
- [ ] All tools accounts created
- [ ] GitHub repos set up and public
- [ ] Gumroad product configured
- [ ] ConvertKit sequences ready
- [ ] Discord server organized
- [ ] Landing page deployed
- [ ] Analytics tracking
- [ ] Support email working

### Test Everything
- [ ] Purchase flow (use Gumroad test mode)
- [ ] Email sequences send
- [ ] Discord invite works
- [ ] GitHub access  granted
- [ ] Landing page loads
- [ ] Forms capture emails
- [ ] Analytics tracks events

### Launch Checklist
- [ ] All content reviewed
- [ ] Pricing confirmed
- [ ] Legal pages added (Terms, Privacy)
- [ ] Payment processor tested
- [ ] Support channels monitored
- [ ] Analytics dashboards ready

---

## 📊 Tool Costs Summary

| **Tool** | **Free Tier** | **Paid Tier** | **When to Upgrade** |
|----------|---------------|---------------|---------------------|
| GitHub | ✅ Unlimited | N/A | Never |
| Gumroad | ✅ 10% fee | 10% fee | Never (fee-based) |
| ConvertKit | ✅ 1k subs | $29/mo @ 1k+ | After 1,000 subs |
| Discord | ✅ Unlimited | $10/mo (perks) | Optional |
| GitHub Pages | ✅ Unlimited | N/A | Never |
| Plausible | Self-host free | $9/mo | If want hosted |
| Domain | N/A | $12/year | Optional |

**Total:** $0/mo initially, ~$50/mo at scale

---

## 🎓 Next Steps

1. Complete this entire setup (4 hours)
2. Test purchase flow end-to-end
3. Create pre-launch content
4. Build email list (100 people before launch)
5. Launch!

---

**All tools are configured! Ready to launch.**

📚 **Related Documents:**
- [Business Plan](./business-plan.md)
- [Launch Checklist](./launch-checklist.md)
- [Marketing Strategy](./marketing-strategy.md)
