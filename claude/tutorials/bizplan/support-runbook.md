# Support Runbook

> **Handle customer support efficiently and professionally**

---

## 🎯 Support Philosophy

**Principle:** Every support interaction is a chance to delight  
**Goal:** Fast, helpful, friendly responses  
**SLA:** Pro: <24h, Free: <48h (best effort)  

---

## 📋 Support Channels

### Channel Priority

| Channel | Response Time | Who | Usage |
|---------|---------------|-----|-------|
| Discord DM (Pro) | < 4 hours | Pro members | Urgent issues |
| Discord #questions | < 12 hours | All | General questions |
| GitHub Issues | < 24 hours | All | Bugs, feature requests |
| Email | < 24 hours | All | Private/sensitive |
| Twitter DM | < 48 hours | All | Quick questions |

---

## 🤝 Support Workflow

### Daily Routine

**Morning (30 mins):**
- [ ] Check Discord mentions/DMs
- [ ] Review GitHub issues
- [ ] Check support email
- [ ] Scan Twitter mentions

**Afternoon (15 mins):**
- [ ] Quick Discord check
- [ ] Urgent issues only

**Evening (30 mins):**
- [ ] Full support sweep
- [ ] Close resolved issues
- [ ] Update FAQ if needed

**Weekly (1 hour):**
- [ ] Review support trends
- [ ] Update documentation
- [ ] Create content from common questions
- [ ] Train community moderators

---

## 💬 Common Questions & Responses

### Category 1: Pre-Purchase

**Q: "Is this worth $99?"**

```
Great question! Here's how I think about it:

✅ Worth it if:
- You're using Restate for a real project
- You want production-ready examples (not just docs)
- You value community support when stuck
- 40+ hours of learning is worth $99 to you

❌ Maybe not if:
- Just casually curious (free modules might be enough)
- Not using Go
- Not building with Restate

Still unsure? Try the 4 free modules first! They're comprehensive and you can always upgrade later.

Also: 30-day money-back guarantee if it's not a good fit.

Let me know if you have specific questions!
```

**Q: "What's the difference between free and Pro?"**

```
Free Tier (Modules 1-4):
✅ Introduction to Restate
✅ Services & handlers basics
✅ State management fundamentals  
✅ Basic workflows
✅ Read-only Discord access

Pro ($99):
✅ Everything in free +
✅ 8 advanced modules (Sagas, Idempotency, Production, etc.)
✅ Solutions repository
✅ Active Discord support
✅ Direct access to me
✅ Lifetime updates

Think of free as "fundamentals" and Pro as "production-ready mastery."

Try free first - if you find it helpful, you'll love Pro!
```

**Q: "Do you offer student discounts?"**

```
Yes! Students get 40% off ($60 instead of $99).

Just reply with:
- Proof of enrollment (student ID or .edu email)
- Your GitHub username

I'll send you a discount code within 24 hours.

Also applies to: bootcamp students, unemployed developers learning, anyone in developing countries.

Learning should be accessible! 🎓
```

### Category 2: Access Issues

**Q: "I purchased but don't have GitHub access"**

```
Hi! Thanks for purchasing!

GitHub invites are sent within 24 hours (usually much faster).

To speed this up:
1. What's your GitHub username?
2. What email did you use for Gumroad?
3. What's your license key?

I'll add you manually right now!

For Discord: Join here [link] and DM me your license key to verify.

Thanks for your patience!
```

**Q: "Discord verification isn't working"**

```
Let's fix this!

Try these steps:
1. Join Discord: [invite link]
2. Go to #verify channel
3. DM me (@yourname) your license key
4. I'll manually assign your @Pro role

License key format: XXXX-XXXX-XXXX
(found in your Gumroad purchase email)

Still stuck? Screenshot what you're seeing and I'll help!
```

**Q: "The code doesn't work / I get an error"**

**Step 1: Gather info**
```
Sorry you're running into issues! Let's debug this.

Can you share:
1. Which module? (e.g., Module 05)
2. What error message? (screenshot or copy/paste)
3. What Go version? (run: go version)
4. What OS? (Mac, Linux, Windows?)

Also - have you:
- Run `go mod tidy`?
- Checked if Restate server is running?
- Tried the example exactly as written first?
```

**Step 2: Common fixes**

```
Most common issues:

1. Restate not running:
docker run -d --name restate_dev -p 8080:8080 -p 9070:9070 restatedev/restate

2. Wrong Go version (need 1.21+):
go version

3. Module not registered:
curl http://localhost:9070/deployments

4. Port conflict:
lsof -i :9090

Try these and let me know what happens!
```

**Step 3: Offer to debug**

```
Still stuck? Let's hop on a quick call or share your screen in Discord.

Or paste:
- Your full error message
- Your full code
- Output of `go mod graph`

I'll figure it out! 🔍
```

### Category 3: Learning Questions

**Q: "I don't understand [concept]"**

```
Great question! [Concept] can be tricky.

Here's how I think about it:

[Simple explanation with analogy]

Example:
[Simple code snippet]

Does that help? Let me know what's still unclear!

Also:
- Video explanation: [YouTube link if exists]
- Related reading: [docs link]
- Similar discussion: [Discord/GitHub link]
```

**Q: "What's the best way to learn this?"**

```
Here's what works best:

1. **Type the code** (don't copy/paste)
   - Typing forces you to read each line
   - You'll notice patterns

2. **Run every example**
   - See it work
   - Break it intentionally to understand errors

3. **Do the exercises**
   - Application solidifies learning
   - Struggle = learning

4. **Build a project**
   - Real application beats theory
   - Start small, iterate

**Timeline:**
- Each module: 30-60 mins
- Full series: 30-50 hours
- Mastery: Build 2-3 projects

Take your time! Distributed systems are complex.

Stuck? Ask in Discord - we've all been there!
```

**Q: "Should I complete modules in order?"**

```
Short answer: Yes!

Each module builds on previous concepts:

Module 01-04: Foundation (free)
  ↓ (must know this first)
Module 05-08: Core patterns
  ↓ (builds on foundation)
Module 09-12: Production/Advanced

Skipping ahead = confusion.

Exception: If you already know basics, start at Module 05.

Where are you now?
```

### Category 4: Refund Requests

**Q: "Can I get a refund?"**

```
Of course! 30-day money-back guarantee, no questions asked.

Before I process it, would you mind sharing:
- What were you hoping for?
- What didn't meet expectations?
- How can I improve?

Your honest feedback helps make this better!

(Refund processing either way - feedback is optional)
```

**Process refund:**
1. Log into Gumroad
2. Customer tab → Find order → Issue refund
3. Send confirmation email:

```
Subject: Refund Processed

Hi,

Refund sent! You'll see it in 5-7 business days.

Thanks for giving it a try. If you ever want to come back, just email me and I'll reinstate your access.

All the best with your Restate journey!

[Your Name]

P.S. If there's anything I can do better, I'm all ears: [email]
```

### Category 5: Feature Requests

**Q: "Can you add a module on [topic]?"**

```
Great idea! I'm always looking for module ideas.

Quick questions:
- Why do you need this?
- What would you want to learn specifically?
- How would you use it in a project?

I track all requests here: [GitHub Discussions/Roadmap]

Current roadmap:
1. Advanced Patterns (in progress)
2. Real-World Case Studies (planned)
3. [Topic] (considering)

Vote/comment there to help prioritize!

Meanwhile, anything else I can help with?
```

---

## 🐛 Bug Report Workflow

### When Someone Reports a Bug

**1. Acknowledge (< 1 hour)**

```
Thanks for reporting! 🐛

Looking into this now. Will update you within 24h.

For tracking: [GitHub issue link]
```

**2. Reproduce (< 4 hours)**

- Try to reproduce locally
- Identify root cause
- Determine severity

**3. Assess Severity:**

**Critical (fix immediately):**
- Code doesn't run at all
- Installation fails
- Security vulnerability

**High (fix this week):**
- Feature broken
- Major confusion in explanation
- Bad example

**Medium (fix this month):**
- Typo
- Formatting issue
- Optimization opportunity

**Low (backlog):**
- Enhancement request
- Nice-to-have improvement

**4. Fix & Notify (< 24 hours for critical)**

```
Fixed! 🎉

Changes:
- [Description of fix]
- Updated in commit: [link]

Pull latest code:
git pull origin main

Let me know if this resolves it!

Thanks for reporting - you helped improve this for everyone! 🙏
```

**5. Follow Up (3 days later)**

```
Quick follow-up: Did the fix work for you?

If so, would you mind testing [related feature] to make sure I didn't break anything? 

Thanks!
```

---

## 😤 Handling Difficult Situations

### Angry Customer

**Example:** "This is terrible! Nothing works! Waste of money!"

**Response:**

```
I'm really sorry you're having a frustrating experience. That's not okay.

Let's fix this right now.

Can you tell me:
1. What specifically isn't working?
2. What error messages are you seeing?
3. What were you trying to do?

I'll prioritize this and get you unstuck today.

Also - if you want a refund while we figure this out, no problem. Just let me know.

```

**Key principles:**
- Empathize first
- Don't get defensive
- Solve the problem quickly
- Offer refund proactively

### Unreasonable Request

**Example:** "Can you build my project for me?"

**Response:**

```
I appreciate you thinking of me!

Unfortunately, I can't build custom projects (bandwidth constraints).

But I can:
✅ Point you to the relevant modules
✅ Answer specific technical questions
✅ Review your code in Discord
✅ Suggest approach/architecture

For custom development, I can recommend:
- [Freelancer platforms]
- [Restate consultants]

What specifically are you stuck on? Maybe I can point you in the right direction?
```

### Scope Creep

**Example:** "Can you teach me Go first?"

**Response:**

```
These tutorials assume basic Go knowledge (structs, interfaces, goroutines).

If you're new to Go, I recommend starting here:
- Tour of Go: https://go.dev/tour
- Go by Example: https://gobyexample.com
- FreeCodeCamp Go Course: [YouTube]

Once you're comfortable with Go basics (2-4 weeks), come back to Restate!

The tutorials will make much more sense with that foundation.

Still want to try? Start with Module 01 - but expect to Google Go syntax along the way!

Let me know how it goes!
```

---

## 📊 Support Metrics

### Track Weekly

- Total support requests
- Average response time
- Resolution time
- Customer satisfaction (ask after resolution)
- Common issues (FAQ candidates)

### Monthly Review

**Questions to ask:**
1. What were the top 3 issues this month?
2. Can documentation prevent these?
3. Are response times meeting SLA?
4. Any patterns in refund requests?
5. What can be automated?

**Actions:**
- Update FAQ
- Improve problematic modules
- Create video for common issues
- Train community moderators

---

## 🤖 Automation Opportunities

### Canned Responses (Discord/Email)

**Setup in Discord:**
1. Install bot with canned responses
2. Create shortcuts:
   - `/access` → GitHub access info
   - `/refund` → Refund policy
   - `/start` → Getting started guide
   - `/bug` → Bug report template

**Setup in Email (Gmail/HelpScout):**
- Templates for common questions
- Keyboard shortcuts

### Chatbot (If Volume Increases)

**When to implement:** > 50 support requests/week

**Options:**
- Discord bot with FAQ
- Website chatbot
- GitHub bot for issues

**What it handles:**
- Common questions
- Links to documentation
- Basic troubleshooting
- Escalates to human when needed

### Knowledge Base

**Create Searchable FAQ:**
- Notion database
- GitHub Wiki
- Dedicated docs site

**Categories:**
- Getting Started
- Purchase & Access
- Technical Issues
- Learning Tips
- Troubleshooting

---

## 👥 Community-Driven Support

### Empower Community to Help

**Strategies:**

1. **Discord Helpers Role**
   - Recognize active helpers
   - Give special role/badge
   - Offer perks (early access, free modules)

2. **Upvote System**
   - Mark helpful answers
   - Build reputation
   - Gamification

3. **Office Hours**
   - Weekly live Q&A
   - Community can help each other
   - You moderate

4. **Peer Review**
   - Students help review each other's code
   - Builds community
   - Reduces your load

---

## ✅ Support Checklist

### Before Responding

- [ ] Read question fully
- [ ] Check if already answered (FAQ, past issues)
- [ ] Reproduce issue if bug
- [ ] Gather needed context

### When Responding

- [ ] Acknowledge quickly
- [ ] Empathize with frustration
- [Provide clear next steps
- [ ] Include relevant links
- [ ] Set expectations (when will you follow up)

### After Resolving

- [ ] Confirm resolution
- [ ] Ask for feedback
- [ ] Update FAQ if needed
- [ ] Thank for reporting

---

## 📚 Templates Library

### Purchase Thank You

```
Subject: Thanks for joining Restate Go Pro! 🎉

Hi!

Super excited to have you!

Your access:
- GitHub: [invitation sent]
- Discord: [link + instructions]
- License: {license_key}

Start here: Module 05

Questions? Reply anytime!

[Your Name]
```

### Bug Fix Notification

```
Subject: Bug Fixed! 🐛→✅

Hi,

Good news - the issue you reported is fixed!

What changed:
[explanation]

Update your code:
git pull origin main

Let me know if you hit any other issues!

Thanks for helping improve this! 🙏

[Your Name]
```

### Re-engagement

```
Subject: Still stuck?

Hey,

Noticed you haven't been active in a while.

Everything okay? Stuck somewhere?

Reply and let me know how I can help!

Still here,
[Your Name]
```

---

**Great support turns customers into advocates. Make every interaction count!**

📚 **Related Documents:**
- [Customer Journey](./customer-journey.md)
- [Automation Workflows](./automation-workflows.md)
- [Business Plan](./business-plan.md)
