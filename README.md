<h1 align="center">
Ars0n Framework v2
</h1>

<p align="center">
  <a href="#about">About</a> •
  <a href="#download-and-install">Download & Install</a> •
  <a href="#how-to-use">How To Use</a> •
  <a href="#troubleshooting">Troubleshooting</a> •
  <a href="#frequently-asked-questions">FAQs</a> •
  <a href="https://www.youtube.com/@rs0n_live" target=”_blank”>YouTube</a> •
  <a href="https://www.linkedin.com/in/harrison-richardson-rs0n-7a55bb158/" target=”_blank”>LinkedIn</a>
</p>

<p align="center">
    <em>🚨 Pre-Alpha Release Out now!!  Beta Launch @ DEFCON 33 Bug Bounty Hunting Village!!! 🚨</em>
</p>

<p align="center">My full bug bounty hunting methodology built into a single framework!  Automate the most common bug bounty hunting workflows and <em>Earn While You Learn</em>!</p>

<p align="center">The goal of this tool is to eliminate the barrier of entry for bug bounty hunting.  My hope is that someone can pick up this tool and start hunting on day one of their AppSec journey 🚀</p>

## About

Howdy!  My name is Harrison Richardson, or `rs0n` (arson) when I want to feel cooler than I really am.  The code in this repository started as a small collection of scripts to help automate many of the common Bug Bounty hunting processes I found myself repeating.  Over time, I built these scripts into an open-source framework that helped thousands of people around the world begin their bug bounty hunting journey.  

However, the first implementation of the framework had a wide range of issues.  The majority of the problems were a result of the tool never being designed with the intent of being shared as an open-source project.  So I got to work on a version 2 that would solve these problems and bring my vision to life!

**The Ars0n Framework V2** is designed to be a tool that allows people to start REAL bug bounty hunting against actual targets on day one!  The framework acts as a wrapper around 20+ widely used bug bounty hunting tools and a clever UI design forces the user into a correct hunting methodology.  It is literally impossible to use this tool without going through rs0n's process!  

The results of each tool are stored in a central database and can be used for understanding/visualizing the target company's attack surface.  Each section also includes a "Help Me Learn!" dropdown that includes a lession plan to help the user understand what part of the methodology they are at, what they are trying to acheive, and most importantly the "Why?" behind it.

My hope is that this modular framework will act as a canvas to help share what I've learned over my career to the next generation of Security Engineers!  Trust me, we need all the help we can get!!

<h4 align="center">
🤠 Did you know that over 95% of scientists believe there is a direct correlation between the amount of coffee I drink and how quickly I can build new features?  Crazy, right?!  Well, now you can test their hypothesis and Buy Me a Coffee through this fancy button!!  🤯
<br>
<br>
<a href="https://www.buymeacoffee.com/rs0n.evolv3" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>
</h4>

<p align="center"><b>Pre-Alpha Demo Videos</b></p>

<div align="center">
  <a href="https://www.youtube.com/watch?v=u-yPpd0UH8w">
    <img src="thumbnail.png" width="250px" alt="Youtube Thumbnail" style="border-radius: 12px;">
  </a>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
  <a href="https://www.youtube.com/watch?v=kAO0stO-hBg">
    <img src="thumbnail2.png" width="250px" alt="Youtube Thumbnail" style="border-radius: 12px;">
  </a>
</div><br>

<p align="center"><b>The Ars0n Framework v2 Includes All These Tools And More!</b></p>

<p align="center">
    <a href="https://github.com/owasp-amass/amass">Amass</a> - Advanced attack surface mapping and asset discovery tool for security research<br>
    <a href="https://github.com/projectdiscovery/subfinder">Subfinder</a> - Fast and reliable subdomain enumeration tool with multiple data sources<br>
    <a href="https://github.com/aboul3la/Sublist3r">Sublist3r</a> - Fast subdomain enumeration tool using various search engines and data sources<br>
    <a href="https://github.com/tomnomnom/assetfinder">Assetfinder</a> - Find assets related to a domain using various data sources and APIs<br>
    <a href="https://github.com/projectdiscovery/httpx">Httpx</a> - Fast and multi-purpose HTTP toolkit for web reconnaissance and scanning<br>
    <a href="https://github.com/jaeles-project/gospider">GoSpider</a> - Fast web spider written in Go for crawling and extracting URLs<br>
    <a href="https://github.com/nsonaniya2010/SubDomainizer">Subdomainizer</a> - Advanced subdomain enumeration tool with multiple discovery methods<br>
    <a href="https://github.com/digininja/CeWL">CeWL</a> - Custom word list generator that spiders websites to create targeted wordlists<br>
    <a href="https://github.com/projectdiscovery/shuffledns">ShuffleDNS</a> - Mass DNS resolver with wildcard filtering and validation capabilities<br>
    <a href="https://github.com/projectdiscovery/nuclei">Nuclei</a> - Fast and customizable vulnerability scanner with extensive template library<br>
    <a href="https://github.com/projectdiscovery/katana">Katana</a> - Fast and powerful web crawler for discovering hidden endpoints and content<br>
    <a href="https://github.com/ffuf/ffuf">FFuf</a> - Fast web fuzzer with support for multiple protocols and advanced filtering<br>
    <a href="https://github.com/lc/gau">GAU</a> - Get All URLs tool that fetches known URLs from various historical data sources<br>
    <a href="https://github.com/pdiscoveryio/ctl">CTL</a> - Certificate Transparency Log tool for discovering subdomains from SSL certificates<br>
    <a href="https://github.com/projectdiscovery/dnsx">DNSx</a> - Fast and multi-purpose DNS toolkit for running multiple DNS queries<br>
    <a href="https://github.com/initstring/cloud_enum">Cloud Enum</a> - Multi-cloud OSINT tool for enumerating public resources in AWS, Azure, and Google Cloud<br>
    <a href="https://github.com/j3ssie/metabigor">Metabigor</a> - OSINT tool for network intelligence gathering including ASN and IP range discovery<br>
    <a href="https://github.com/gwen001/github-search">GitHub Recon</a> - GitHub reconnaissance tool for discovering organization mentions and domain patterns<br>
    <a href="https://github.com/projectdiscovery/naabu">Naabu</a> - Fast port scanner for discovering open ports and services<br>
    <a href="https://github.com/whoxy/whoxy">Reverse Whois</a> - Reverse WHOIS lookup using Whoxy to find domains registered by the same entity<br>
    <a href="https://securitytrails.com">SecurityTrails</a> - Comprehensive DNS, domain, and IP data provider for digital asset discovery<br>
    <a href="https://censys.io">Censys</a> - Internet-wide scanning platform for discovering and monitoring assets<br>
    <a href="https://shodan.io">Shodan</a> - Search engine for internet-connected devices and services<br>
</p>

## Download And Install

This framework consists of 20+ Docker containers along w/ a Docker Compose Manifest to automate the process of deploying these containers.

1. Download the Zip File for the <a href="https://github.com/R-s0n/ars0n-framework-v2/releases/download/beta-test/ars0n-framework-v2-beta-0.0.0.zip">latest release</a>
2. Unzip the files
3. Navigate to the directory with the `docker-compose.yml` file
4. Run `docker-compose up --build`

*HINT: If you get a docker error, the problem is probably w/ docker, not my framework*

### Windows

**Step 1:** Download the framework
```powershell
Invoke-WebRequest -Uri "https://github.com/R-s0n/ars0n-framework-v2/releases/download/beta-test/ars0n-framework-v2-beta-0.0.0.zip" -OutFile "ars0n-framework-v2.zip"
```

**Step 2:** Extract the zip file
```powershell
Expand-Archive -Path "ars0n-framework-v2.zip" -DestinationPath "."
```

**Step 3:** Navigate to the framework directory
```powershell
cd ars0n-framework-v2
```

**Step 4:** Start the framework
```powershell
docker-compose up --build
```

### Mac

**Step 1:** Download the framework
```bash
curl -L -o ars0n-framework-v2.zip "https://github.com/R-s0n/ars0n-framework-v2/releases/download/beta-test/ars0n-framework-v2-beta-0.0.0.zip"
```

**Step 2:** Extract the zip file
```bash
unzip ars0n-framework-v2.zip
```

**Step 3:** Navigate to the framework directory
```bash
cd ars0n-framework-v2
```

**Step 4:** Start the framework
```bash
docker-compose up --build
```

### Linux

**Step 1:** Download the framework
```bash
wget "https://github.com/R-s0n/ars0n-framework-v2/releases/download/beta-0.0.0/ars0n-framework-v2-beta-0.0.0.zip"
```

**Step 2:** Extract the zip file
```bash
unzip ars0n-framework-v2-beta-0.0.0.zip
```

**Step 3:** Navigate to the framework directory
```bash
cd ars0n-framework-v2
```

**Step 4:** Start the framework
```bash
docker-compose up --build
```

## How To Use

Coming Soon...

## Troubleshooting

Coming Soon...

## Frequently Asked Questions

Coming Soon...

## License

This project is licensed under the GNU General Public License v3.0 (GPL-3.0). This means:

- You can freely use, modify, and distribute this software
- If you distribute modified versions, you must:
  - Make your source code available
  - Include the original copyright notice
  - Use the same license (GPL-3.0)
  - Document your changes

For more details, see the [LICENSE](LICENSE) file in the repository.

<p align="right">~ by rs0n w/ ❤️</p>
<p align="center"><em>Copyright (C) 2025 Arson Security, LLC</em></p>
