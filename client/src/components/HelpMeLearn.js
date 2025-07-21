import React from 'react';
import { Accordion, ListGroup } from 'react-bootstrap';

const HelpMeLearn = ({ section }) => {
  const sections = {
    amass: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What stage of the methodology are we at and what are we trying to accomplish?",
          answers: [
            "This workflow is part of the Reconnaissance (Recon) phase of the Bug Bounty Hunting methodology.",
            "We have identified a root domain that belongs to the target organization. Now our goal is to find a list of subdomains for that root domain that point to a live web server. Each live web server is a possible target for bug bounty testing. At the end of this workflow, we will have a list of Target URLs that can be added as \"URL\" Scope Targets."
          ]
        },
        {
          question: "What is Amass and how does it work?",
          answers: [
            "Amass is a powerful open-source tool for performing attack surface mapping and external asset discovery. It uses various techniques including DNS enumeration, web scraping, and data source integration to build a comprehensive map of an organization's external attack surface.",
            "The tool works by combining multiple data sources and techniques: DNS enumeration, web scraping, certificate transparency logs, and various third-party data sources. It systematically discovers subdomains, IP addresses, and other assets associated with the target domain while respecting rate limits and avoiding detection."
          ]
        },
        {
          question: "How do I read the Amass output?",
          answers: [
            "Scan History shows the time, date, and results of previous scans. This helps track your reconnaissance progress and compare results across different scans.",
            "Raw Results shows the complete output of the Amass scan, including all discovered subdomains, IP addresses, and associated metadata. This is useful for detailed analysis and verification.",
            "DNS Records provides detailed DNS information for discovered subdomains, including A records, CNAME records, and other DNS configurations that help understand the infrastructure.",
            "Infrastructure View shows a comprehensive overview of the target's infrastructure, including cloud services, hosting providers, and other technical details about the discovered assets."
          ]
        }
      ]
    },
    subdomainScraping: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What are subdomain scraping tools and why do we need them?",
          answers: [
            "Subdomain scraping tools use various techniques to discover subdomains from public sources, web scraping, and third-party data. They complement Amass by finding additional subdomains that might have been missed.",
            "Each tool has its own strengths: Httpx finds live web servers, Gau discovers URLs from JavaScript files, Sublist3r uses multiple search engines, Assetfinder focuses on DNS enumeration, and CTL checks certificate transparency logs."
          ]
        },
        {
          question: "How do I use these tools effectively?",
          answers: [
            "Start with Httpx to identify live web servers, then use Gau to discover URLs from JavaScript files. Follow up with Sublist3r for search engine results, Assetfinder for DNS enumeration, and CTL for certificate transparency logs.",
            "After running each tool, review the results in their respective modals. Use the Consolidate button to combine all discovered subdomains into a single list, then run Httpx again to verify which ones are live web servers."
          ]
        }
      ]
    },
    bruteForce: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is subdomain brute-forcing and why is it important?",
          answers: [
            "Subdomain brute-forcing is a technique that systematically tries different subdomain names against a domain to discover valid subdomains. This method can find subdomains that weren't discovered through passive reconnaissance or public sources.",
            "While this technique is more aggressive than passive methods, it's crucial for finding hidden or forgotten subdomains that might be vulnerable. It's particularly useful for discovering development, staging, or internal subdomains that might not be publicly advertised."
          ]
        },
        {
          question: "How do I use the brute-force tools effectively?",
          answers: [
            "Start with Subfinder for initial enumeration, then use ShuffleDNS for DNS-based brute-forcing. Follow up with CeWL to generate custom wordlists based on the target's content, and finally use GoSpider for crawling and discovering additional subdomains.",
            "After running each tool, review the results in their respective modals. Use the Consolidate button to combine all discovered subdomains into a single list, then run Httpx again to verify which ones are live web servers. This ensures you have a comprehensive list of valid subdomains."
          ]
        }
      ]
    },
    javascriptDiscovery: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is JavaScript/Link Discovery and why is it important?",
          answers: [
            "JavaScript/Link Discovery is a technique that analyzes web applications' JavaScript files and HTML content to find hidden subdomains, endpoints, and other assets. This method is particularly effective because modern web applications often contain valuable information in their client-side code.",
            "This approach can discover subdomains that aren't visible through DNS enumeration or public sources, as they might be dynamically loaded or referenced in JavaScript code."
          ]
        },
        {
          question: "How do I use these tools effectively?",
          answers: [
            "Start with GoSpider to crawl the target's web applications and discover JavaScript files and links. Follow up with Subdomainizer to extract subdomains from JavaScript files and other web content. Finally, use Nuclei Screenshot to capture visual evidence of discovered assets.",
            "After running each tool, review the results in their respective modals. Use the Consolidate button to combine all discovered subdomains into a single list, then run Httpx again to verify which ones are live web servers. This ensures you have a comprehensive list of valid subdomains."
          ]
        }
      ]
    },
    decisionPoint: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is the Decision Point and why is it important?",
          answers: [
            "The Decision Point is where you evaluate all the reconnaissance results and decide which discovered assets should be added as URL Scope Targets. This is a crucial step as it determines which assets you'll focus on during your bug bounty testing.",
            "At this stage, you should have a comprehensive list of live web servers from various discovery methods: Amass enumeration, subdomain scraping, brute-forcing, and JavaScript analysis. The Decision Point helps you organize and prioritize these assets for testing."
          ]
        },
        {
          question: "How do I evaluate and select targets effectively?",
          answers: [
            "Start by reviewing the consolidated list of discovered subdomains. Use the ROI Report to identify high-value targets based on factors like technology stack, security headers, and potential attack surface. Pay special attention to assets that might contain sensitive information or critical functionality.",
            "After identifying promising targets, use the 'Add URL Scope Target' button to add them to your scope. Consider factors like the target's importance to the organization, potential impact of vulnerabilities, and your testing priorities when selecting targets."
          ]
        }
      ]
    },
    companyNetworkRanges: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What stage of the methodology are we at and what are we trying to accomplish?",
          answers: [
            "This workflow is part of the Reconnaissance (Recon) phase of the Bug Bounty Hunting methodology, specifically focused on company-wide asset discovery.",
            "We have identified a target company and now our goal is to discover their complete attack surface, including on-premises infrastructure, network ranges, and all associated domains. This approach is more comprehensive than targeting a single domain and helps identify the full scope of the organization's digital assets."
          ]
        },
        {
          question: "What are ASN and network range discovery tools and why are they important?",
          answers: [
            "ASN (Autonomous System Number) and network range discovery tools help identify the complete network infrastructure belonging to a target organization. These tools discover IP ranges, subnets, and network blocks that the company owns or controls.",
            "This information is crucial because it reveals the organization's on-premises infrastructure, data centers, and network boundaries. Understanding the full network scope helps identify potential entry points and attack vectors that might not be visible through domain-based reconnaissance alone."
          ]
        },
        {
          question: "How do I use Amass Intel and Metabigor effectively?",
          answers: [
            "Start with Amass Intel to gather comprehensive intelligence about the target organization's network infrastructure, including ASN information, IP ranges, and associated domains. This provides a broad overview of the organization's digital footprint.",
            "Follow up with Metabigor to perform additional OSINT gathering and discover additional network ranges, subnets, and infrastructure details. After running both tools, use the Consolidate button to combine all discovered network ranges into a single list for further processing."
          ]
        }
      ]
    },
    companyLiveWebServers: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is the purpose of discovering live web servers in company infrastructure?",
          answers: [
            "Discovering live web servers within a company's infrastructure helps identify potential attack vectors and entry points that might not be publicly advertised. These servers could include internal applications, development environments, admin panels, or legacy systems.",
            "This step is crucial for understanding the organization's complete attack surface, as on-premises infrastructure often contains critical business applications, sensitive data, or systems that may have weaker security controls than public-facing assets."
          ]
        },
        {
          question: "How does the network range processing workflow work?",
          answers: [
            "First, use Trim Network Ranges to remove any invalid or overly broad ranges that might cause scanning issues. This helps focus the scan on legitimate company infrastructure and reduces false positives.",
            "Next, use Consolidate to combine all discovered network ranges into a single list. Then run IP/Port Scan to identify live hosts and open ports within these ranges. Finally, use Gather Metadata to collect detailed information about discovered web servers, including technology stack, headers, and potential vulnerabilities."
          ]
        },
        {
          question: "What should I look for in the results?",
          answers: [
            "Focus on web servers that might contain sensitive information, such as admin panels, development environments, or internal applications. Look for servers with unusual ports, outdated technology stacks, or missing security headers.",
            "Pay attention to servers that might be misconfigured, such as those exposing internal services, development tools, or administrative interfaces. These often represent high-value targets for bug bounty testing."
          ]
        }
      ]
    },
    companyRootDomainDiscovery: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is root domain discovery and why is it important for company reconnaissance?",
          answers: [
            "Root domain discovery involves finding all the primary domains that belong to a target organization. This is crucial because companies often own multiple domains for different purposes, subsidiaries, or acquisitions that may not be immediately obvious.",
            "Discovering all root domains helps build a comprehensive picture of the organization's digital presence and identifies potential attack surfaces that might be overlooked when focusing on a single domain."
          ]
        },
        {
          question: "How do the no-API-key tools work and what are their strengths?",
          answers: [
            "Google Dorking uses advanced search operators to find company domains, subdomains, and exposed information through search engine results. This technique can discover domains mentioned in public documents, job postings, or other online content.",
            "CRT (Certificate Transparency) searches certificate transparency logs to find domains that have SSL certificates, revealing domains that might not be publicly advertised. Reverse WHOIS looks up domain registration information to find other domains registered by the same entity or contact information."
          ]
        },
        {
          question: "How do I use these tools effectively?",
          answers: [
            "Start with Google Dorking using company-specific search terms and operators to find domains mentioned in public sources. Follow up with CRT to discover domains with SSL certificates, and use Reverse WHOIS to find domains registered by the same organization.",
            "After running each tool, review the results and add legitimate company domains to your scope. Use the Consolidate button to combine all discovered domains, then proceed to subdomain enumeration for each root domain to build a comprehensive attack surface."
          ]
        }
      ]
    },
    companyRootDomainDiscoveryAPI: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What are the API-based root domain discovery tools and why do they require API keys?",
          answers: [
            "API-based tools provide access to specialized databases and services that contain comprehensive domain and infrastructure information. These tools require API keys because they access premium data sources that aren't publicly available.",
            "SecurityTrails provides DNS, domain, and IP data from their extensive database. GitHub Recon searches public repositories for organization mentions and domain patterns. Shodan searches for internet-connected devices and services. Censys provides certificate and domain data from their scanning infrastructure."
          ]
        },
        {
          question: "How do I configure and use these tools effectively?",
          answers: [
            "First, configure your API keys in the Settings modal. Each tool requires specific API credentials that you can obtain from their respective websites. Once configured, these tools will be enabled and ready to use.",
            "Run each tool to discover additional domains and infrastructure information. These tools often find domains and assets that aren't discoverable through public sources, providing a more comprehensive view of the organization's attack surface."
          ]
        },
        {
          question: "What should I look for in the API tool results?",
          answers: [
            "Look for domains that might represent different business units, subsidiaries, or acquisitions. Pay attention to domains that might contain sensitive applications, such as admin portals, development environments, or internal services.",
            "Focus on domains that might be overlooked or forgotten, as these often have weaker security controls. Also look for patterns in domain naming conventions that might reveal additional undiscovered assets."
          ]
        }
      ]
    },
    companySubdomainEnumeration: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is company-wide subdomain enumeration and how does it differ from single-domain enumeration?",
          answers: [
            "Company-wide subdomain enumeration involves discovering subdomains across all the organization's root domains, not just a single domain. This approach provides a comprehensive view of the organization's complete attack surface.",
            "This differs from single-domain enumeration because it requires coordinating multiple scans across different domains, managing larger datasets, and identifying patterns across the organization's infrastructure. The goal is to find all web applications and services across the entire company."
          ]
        },
        {
          question: "How do the company subdomain enumeration tools work?",
          answers: [
            "Amass Enum Company performs subdomain enumeration across multiple domains simultaneously, using the same techniques as single-domain Amass but scaled for company-wide discovery. DNSx Company performs DNS resolution and validation across all discovered subdomains.",
            "Katana Company crawls web applications across all company domains to discover additional subdomains, endpoints, and assets through JavaScript analysis and link discovery. These tools work together to build a comprehensive map of the organization's web infrastructure."
          ]
        },
        {
          question: "How do I manage and prioritize the results from company-wide enumeration?",
          answers: [
            "Start by reviewing the consolidated results from all tools to identify high-value targets. Look for subdomains that might contain sensitive applications, admin interfaces, or development environments.",
            "Use the ROI Report to prioritize targets based on factors like technology stack, security headers, and potential attack surface. Focus on assets that might contain critical business functionality or sensitive data, as these often represent the highest-value targets for bug bounty testing."
          ]
        }
      ]
    },
    companyDecisionPoint: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is the Company Decision Point and how does it differ from the Wildcard Decision Point?",
          answers: [
            "The Company Decision Point is where you evaluate all the reconnaissance results from the company-wide discovery process and decide which discovered assets should be added as scope targets. This differs from the Wildcard Decision Point because it involves evaluating a much larger and more diverse set of assets.",
            "At this stage, you should have discovered network ranges, root domains, subdomains, and live web servers across the entire organization. The Company Decision Point helps you organize, prioritize, and select the most promising targets from this comprehensive attack surface."
          ]
        },
        {
          question: "How do I evaluate and prioritize company-wide assets effectively?",
          answers: [
            "Start by categorizing discovered assets by type: on-premises infrastructure, public web applications, internal services, and development environments. Consider the potential impact and likelihood of finding vulnerabilities in each category.",
            "Use the ROI Report to identify high-value targets based on technology stack, security posture, and business criticality. Pay special attention to assets that might contain sensitive data, critical business functionality, or represent entry points to internal networks."
          ]
        },
        {
          question: "What strategies should I use for selecting company-wide scope targets?",
          answers: [
            "Consider both breadth and depth in your target selection. Include a mix of public-facing applications, internal services, and infrastructure components to maximize your chances of finding vulnerabilities.",
            "Focus on assets that might be overlooked or have weaker security controls, such as development environments, admin interfaces, or legacy systems. Also consider the potential for chaining vulnerabilities across different assets within the organization."
          ]
        }
      ]
    },
    companyNucleiScanning: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What stage of the methodology are we at and what are we trying to accomplish?",
          answers: [
            "This is the Vulnerability Assessment phase of the Bug Bounty Hunting methodology.",
            "We have completed reconnaissance and discovered the company's attack surface including live web servers and subdomains.",
            "Now our goal is to identify security vulnerabilities and potential bug bounty targets using automated scanning tools."
          ]
        },
        {
          question: "What is Nuclei and how does it help with bug bounty hunting?",
          answers: [
            "Nuclei is a fast vulnerability scanner that uses YAML-based templates to identify security issues.",
            "It can scan for thousands of known vulnerabilities across web applications, APIs, and infrastructure.",
            "Nuclei templates are community-driven and constantly updated with the latest security findings.",
            "It's particularly effective for finding common web vulnerabilities like XSS, SQL injection, and misconfigurations."
          ]
        },
        {
          question: "How do we configure Nuclei for effective company-wide scanning?",
          answers: [
            "Select Target Assets: Choose which discovered subdomains and web servers to scan based on your scope.",
            "Choose Templates: Select appropriate vulnerability templates based on the target technology stack.",
            "Configure Scan Parameters: Set rate limits, timeouts, and other settings to avoid overwhelming targets.",
            "Review Scan Results: Analyze findings to identify high-impact vulnerabilities for bug bounty submission."
          ]
        },
        {
          question: "What should we look for in Nuclei scan results?",
          answers: [
            "High-Severity Findings: Critical and high-impact vulnerabilities that could lead to significant security breaches.",
            "Common Web Vulnerabilities: XSS, SQL injection, CSRF, and other OWASP Top 10 issues.",
            "Misconfigurations: Security headers, SSL/TLS issues, and server configuration problems.",
            "Information Disclosure: Sensitive data exposure, error messages, and debugging information."
          ]
        }
      ]
    },
    companyConsolidateRootDomains: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is the purpose of consolidating root domains in company reconnaissance?",
          answers: [
            "Consolidating root domains is a crucial step that combines all discovered domains from various sources into a single, deduplicated list.",
            "This process ensures we have a comprehensive view of all the organization's digital assets without duplicates or invalid entries.",
            "The consolidated list becomes the foundation for systematic subdomain enumeration across the entire company's attack surface."
          ]
        },
        {
          question: "How does the consolidation workflow work?",
          answers: [
            "First, use Trim Root Domains to remove any invalid, duplicate, or irrelevant domains that might cause issues during scanning.",
            "Next, use Consolidate to combine all discovered domains from different tools into a single list, removing duplicates.",
            "Then use Investigate to gather additional information about each domain to verify they belong to the target organization.",
            "Finally, use Add Wildcard Target to convert verified domains into Wildcard scope targets for subdomain enumeration."
          ]
        },
        {
          question: "What should I look for when evaluating consolidated domains?",
          answers: [
            "Focus on domains that are confirmed to belong to the target organization through investigation and validation.",
            "Prioritize domains that might contain sensitive applications, such as admin portals, development environments, or internal services.",
            "Look for patterns in domain naming conventions that might reveal additional undiscovered assets or business units.",
            "Consider the potential impact and scope of each domain when deciding which ones to add as Wildcard targets."
          ]
        }
      ]
    },
    companyBruteForceCrawl: {
      title: "Help Me Learn!",
      items: [
        {
          question: "What is cloud asset enumeration through brute-force and crawling techniques?",
          answers: [
            "Cloud asset enumeration involves discovering cloud-based resources and services that belong to the target organization.",
            "Brute-force techniques systematically try different combinations of service names, regions, and configurations to find cloud assets.",
            "Crawling techniques analyze web applications and JavaScript files to discover cloud endpoints, APIs, and services that might not be publicly advertised."
          ]
        },
        {
          question: "How do Cloud Enum and Katana work together for comprehensive discovery?",
          answers: [
            "Cloud Enum specializes in discovering cloud infrastructure across AWS, Azure, and Google Cloud using brute-force techniques.",
            "It tries common service names, bucket names, and cloud resource patterns to find misconfigured or exposed cloud assets.",
            "Katana performs intelligent web crawling to discover cloud endpoints, APIs, and services referenced in web applications.",
            "Together, they provide both infrastructure-level and application-level cloud asset discovery for a complete picture."
          ]
        },
        {
          question: "What types of cloud assets should I focus on for bug bounty hunting?",
          answers: [
            "Misconfigured Cloud Storage: S3 buckets, Azure blobs, or Google Cloud Storage that might be publicly accessible.",
            "Cloud APIs and Endpoints: Services that might have weak authentication or authorization controls.",
            "Development and Staging Environments: Cloud resources used for testing that might have weaker security controls.",
            "Cloud Management Interfaces: Admin panels or configuration interfaces that might be exposed or misconfigured."
          ]
        },
        {
          question: "How do I prioritize cloud assets for vulnerability testing?",
          answers: [
            "Focus on assets that contain sensitive data or critical business functionality.",
            "Prioritize misconfigured or publicly accessible cloud resources that might be overlooked.",
            "Look for cloud assets that might provide access to internal networks or other sensitive systems.",
            "Consider the potential impact of vulnerabilities in cloud infrastructure, as they often affect multiple services."
          ]
        }
      ]
    }
  };

  const currentSection = sections[section];

  return (
    <Accordion data-bs-theme="dark" className="mb-3">
      <Accordion.Item eventKey="0">
        <Accordion.Header className="fs-5">{currentSection.title}</Accordion.Header>
        <Accordion.Body className="bg-dark">
          <ListGroup as="ul" variant="flush">
            {currentSection.items.map((item, index) => (
              <ListGroup.Item key={index} as="li" className="bg-dark text-danger 5">
                {item.question}
                <ListGroup as="ul" variant="flush" className="mt-2">
                  {item.answers.map((answer, answerIndex) => (
                    <ListGroup.Item key={answerIndex} as="li" className="bg-dark text-white fst-italic fs-6">
                      {answer}{' '}
                      <a href="#" className="text-danger text-decoration-none">
                        Learn More
                      </a>
                    </ListGroup.Item>
                  ))}
                </ListGroup>
              </ListGroup.Item>
            ))}
          </ListGroup>
        </Accordion.Body>
      </Accordion.Item>
    </Accordion>
  );
};

export default HelpMeLearn; 