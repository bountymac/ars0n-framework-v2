export const lessons = {
  reconnaissancePhase: {
    title: "Bug Bounty Methodology: Reconnaissance Phase",
    overview: "The Reconnaissance phase is the foundation of successful bug bounty hunting. During this phase, we systematically map out the target organization's digital footprint to identify potential attack vectors and vulnerable assets.",
    sections: [
      {
        title: "Understanding the Reconnaissance Phase",
        icon: "fa-search",
        content: [
          "Reconnaissance is the first and most critical phase of bug bounty hunting. It involves gathering as much information as possible about your target organization without directly interacting with their systems in a way that could be detected or cause harm.",
          "The goal is to build a comprehensive map of the organization's digital assets, including domains, subdomains, IP ranges, network infrastructure, and services. This intelligence forms the foundation for all subsequent testing activities.",
          "Effective reconnaissance often determines the success of your entire bug bounty engagement. The more thorough your reconnaissance, the more potential targets you'll discover, increasing your chances of finding vulnerabilities."
        ],
        keyPoints: [
          "Reconnaissance is primarily passive - avoiding detection while gathering information",
          "The phase focuses on discovering assets, not testing them for vulnerabilities",
          "Information gathered here guides all future testing decisions",
          "Thorough reconnaissance often reveals assets that organizations have forgotten about"
        ]
      },
      {
        title: "On-Premises vs. Cloud Infrastructure",
        icon: "fa-server",
        content: [
          "Modern organizations typically have a hybrid infrastructure consisting of both on-premises and cloud-based assets. Understanding this distinction is crucial for comprehensive reconnaissance.",
          "On-premises infrastructure refers to servers, applications, and services hosted in the organization's own data centers or physical locations. These assets are often connected to the internet through the organization's own network ranges and ASNs.",
          "Cloud infrastructure, by contrast, is hosted by third-party providers like AWS, Azure, or Google Cloud. These assets may not be immediately obvious through traditional network-based reconnaissance techniques."
        ],
        keyPoints: [
          "On-premises assets are typically accessed through the organization's own IP ranges",
          "Cloud assets may be hosted on shared infrastructure with other organizations",
          "On-premises assets often have different security postures than cloud assets",
          "Legacy on-premises systems may have weaker security controls"
        ]
      },
      {
        title: "From Company Name to Network Ranges",
        icon: "fa-route",
        content: [
          "The process of going from a company name to network ranges involves several steps and data sources. This progression allows us to map the organization's network footprint systematically.",
          "We start with publicly available information about the organization, including business registrations, WHOIS data, and regulatory filings. This information often reveals subsidiary companies, acquisition history, and business relationships.",
          "Next, we use specialized tools and databases to discover the organization's Autonomous System Numbers (ASNs) and associated network ranges. These technical identifiers reveal the IP address space that the organization controls or uses."
        ],
        keyPoints: [
          "Company information often reveals subsidiaries and acquisitions",
          "ASN data provides authoritative information about network ownership",
          "Network ranges define the IP address space where on-premises assets are likely located",
          "This process reveals infrastructure that may not be discoverable through domain-based reconnaissance"
        ]
      }
    ],
    practicalTips: [
      "Always start with broad reconnaissance before narrowing down to specific targets",
      "Document everything - reconnaissance data becomes invaluable for future engagements",
      "Use multiple data sources to validate and cross-reference your findings",
      "Pay special attention to assets that seem forgotten or unmaintained",
      "Consider the organization's business model when planning reconnaissance - different industries have different infrastructure patterns",
      "Remember that reconnaissance is an iterative process - new information often reveals additional targets"
    ],
    furtherReading: [
      {
        title: "OWASP Testing Guide - Information Gathering",
        url: "https://owasp.org/www-project-web-security-testing-guide/v42/4-Web_Application_Security_Testing/01-Information_Gathering/",
        description: "Comprehensive guide to web application reconnaissance techniques"
      },
      {
        title: "NIST Cybersecurity Framework",
        url: "https://www.nist.gov/cyberframework",
        description: "Understanding how organizations structure their cybersecurity programs"
      }
    ]
  },

  asnNetworkRanges: {
    title: "ASNs and Network Ranges in Bug Bounty Hunting",
    overview: "Autonomous System Numbers (ASNs) and network ranges are fundamental concepts in internet infrastructure that provide crucial intelligence for bug bounty hunters seeking to understand an organization's complete attack surface.",
    sections: [
      {
        title: "Understanding Autonomous System Numbers (ASNs)",
        icon: "fa-network-wired",
        content: [
          "An Autonomous System Number (ASN) is a unique identifier assigned to networks that operate under a single administrative domain. Think of it as a 'license plate' for networks on the internet.",
          "ASNs are assigned by Regional Internet Registries (RIRs) and are used in Border Gateway Protocol (BGP) routing to determine how traffic flows between different networks on the internet.",
          "For bug bounty hunters, ASNs are valuable because they provide authoritative information about which IP address ranges belong to which organizations. This is often more reliable than WHOIS data for individual IP addresses."
        ],
        keyPoints: [
          "ASNs are globally unique identifiers for networks",
          "They are assigned by Regional Internet Registries (ARIN, RIPE, APNIC, etc.)",
          "ASNs are used in BGP routing to connect networks",
          "Organizations can own multiple ASNs for different purposes or regions"
        ],
        examples: [
          {
            code: "AS15169 - Google LLC",
            description: "Google's primary ASN"
          },
          {
            code: "AS32934 - Facebook, Inc.",
            description: "Meta's (formerly Facebook) primary ASN"
          },
          {
            code: "AS8075 - Microsoft Corporation",
            description: "Microsoft's primary ASN"
          }
        ]
      },
      {
        title: "Network Ranges and CIDR Notation",
        icon: "fa-sitemap",
        content: [
          "Network ranges define blocks of IP addresses that belong to an organization. These ranges are typically expressed in CIDR (Classless Inter-Domain Routing) notation, which specifies both the network address and the number of bits used for the network portion.",
          "For example, 192.168.1.0/24 represents a network with 256 IP addresses (192.168.1.0 through 192.168.1.255), where the first 24 bits identify the network and the last 8 bits identify individual hosts.",
          "Organizations may own multiple network ranges of different sizes, depending on their infrastructure needs. Large organizations often have Class A or Class B networks, while smaller organizations might have smaller ranges."
        ],
        keyPoints: [
          "CIDR notation expresses both network address and subnet mask",
          "The /X notation indicates how many bits are used for the network portion",
          "Larger organizations typically have larger network ranges",
          "Network ranges can be subdivided into smaller subnets"
        ],
        examples: [
          {
            code: "8.8.8.0/24",
            description: "Google's public DNS network range (256 addresses)"
          },
          {
            code: "157.240.0.0/16",
            description: "Facebook network range (65,536 addresses)"
          },
          {
            code: "20.0.0.0/8",
            description: "Microsoft Azure cloud network (16,777,216 addresses)"
          }
        ]
      },
      {
        title: "Why ASNs and Network Ranges Matter for Bug Bounty Hunting",
        icon: "fa-bullseye",
        content: [
          "Understanding ASNs and network ranges is crucial for bug bounty hunters because it reveals the complete attack surface of an organization, not just their public-facing domains.",
          "Many organizations have internal services, development environments, admin panels, and legacy systems running on their network ranges that aren't linked from public websites or indexed by search engines.",
          "These 'hidden' assets often have weaker security controls because they were intended for internal use only, making them prime targets for security researchers.",
          "By mapping an organization's complete network footprint, bug bounty hunters can discover assets that competitors might miss, leading to unique findings and higher bounty payouts."
        ],
        keyPoints: [
          "Network ranges reveal assets beyond public-facing domains",
          "Internal services often have weaker security controls",
          "Comprehensive network mapping leads to unique target discovery",
          "ASN data provides authoritative ownership information"
        ]
      },
      {
        title: "Regional Internet Registries and Data Sources",
        icon: "fa-globe",
        content: [
          "Regional Internet Registries (RIRs) are organizations responsible for allocating IP addresses and ASNs within specific geographic regions. Understanding these organizations helps bug bounty hunters know where to find authoritative information.",
          "The five RIRs are: ARIN (North America), RIPE NCC (Europe and Middle East), APNIC (Asia-Pacific), LACNIC (Latin America and Caribbean), and AFRINIC (Africa).",
          "Each RIR maintains databases of IP address allocations and ASN assignments that can be queried to find information about network ownership."
        ],
        keyPoints: [
          "ARIN covers North America",
          "RIPE NCC covers Europe, Middle East, and parts of Central Asia",
          "APNIC covers Asia-Pacific region",
          "LACNIC covers Latin America and Caribbean",
          "AFRINIC covers Africa"
        ]
      }
    ],
    practicalTips: [
      "Use multiple ASN lookup tools to cross-reference your findings",
      "Look for patterns in ASN assignments that might reveal subsidiary relationships",
      "Pay attention to the age of ASN assignments - older ASNs often have more interesting legacy infrastructure",
      "Consider geographical distribution - multinational companies often have ASNs in multiple regions",
      "Don't ignore small network ranges - they sometimes contain the most interesting assets",
      "Remember that organizations can lease IP space from other providers, so ownership isn't always straightforward"
    ],
    furtherReading: [
      {
        title: "ARIN WHOIS Database",
        url: "https://whois.arin.net/",
        description: "North American registry for IP addresses and ASNs"
      },
      {
        title: "RIPE Database",
        url: "https://apps.db.ripe.net/db-web-ui/",
        description: "European registry for IP addresses and ASNs"
      },
      {
        title: "BGP Toolkit",
        url: "https://bgp.tools/",
        description: "Tools for analyzing BGP routing and ASN information"
      }
    ]
  },

  amassIntelMetabigor: {
    title: "Amass Intel and Metabigor: OSINT Tools for Infrastructure Discovery",
    overview: "Amass Intel and Metabigor are specialized Open Source Intelligence (OSINT) tools designed to discover and map organizational network infrastructure through automated querying of public databases and registries.",
    sections: [
      {
        title: "Amass Intel: Intelligence Gathering Framework",
        icon: "fa-brain",
        content: [
          "Amass Intel is part of the OWASP Amass project, specifically designed for gathering intelligence about organizations and their network infrastructure. Unlike Amass Enum which focuses on subdomain enumeration, Amass Intel concentrates on organizational intelligence.",
          "The tool queries multiple data sources including WHOIS databases, Regional Internet Registries (RIRs), routing databases, and certificate transparency logs to build a comprehensive picture of an organization's network footprint.",
          "Amass Intel can discover ASNs associated with an organization, IP address ranges allocated to those ASNs, and related domains and subdomains that might not be discoverable through traditional DNS enumeration."
        ],
        keyPoints: [
          "Part of the OWASP Amass project focusing on organizational intelligence",
          "Queries authoritative sources like RIRs and routing databases",
          "Discovers ASNs, IP ranges, and associated domains",
          "Provides more reliable data than passive DNS sources alone"
        ],
        examples: [
          {
            code: "amass intel -d example.com",
            description: "Basic intelligence gathering for example.com"
          },
          {
            code: "amass intel -org 'Example Corporation'",
            description: "Intelligence gathering using organization name"
          },
          {
            code: "amass intel -asn 12345",
            description: "Gathering information about a specific ASN"
          }
        ]
      },
      {
        title: "Metabigor: Multi-Source OSINT Intelligence",
        icon: "fa-search-plus",
        content: [
          "Metabigor is a specialized OSINT tool that focuses on discovering network ranges and infrastructure information through multiple intelligence gathering techniques. The name combines 'Meta' (beyond) and 'Bigor' (a play on 'bigger'), reflecting its goal of finding comprehensive intelligence.",
          "The tool searches through various public databases, routing registries, and internet registries to find IP ranges, subnets, and network blocks associated with target organizations.",
          "Metabigor is particularly effective at discovering infrastructure that organizations might not publicly advertise, including legacy network ranges, acquired infrastructure, and subsidiary networks."
        ],
        keyPoints: [
          "Specialized tool for network range and infrastructure discovery",
          "Queries multiple public databases and registries",
          "Effective at finding non-obvious or legacy infrastructure",
          "Can discover subsidiary and acquisition-related networks"
        ],
        examples: [
          {
            code: "metabigor net -q 'Example Corp'",
            description: "Network range discovery for Example Corp"
          },
          {
            code: "metabigor net -q 'AS12345'",
            description: "Network ranges associated with a specific ASN"
          }
        ]
      },
      {
        title: "Data Sources and Methodologies",
        icon: "fa-database",
        content: [
          "Both tools leverage multiple authoritative data sources to ensure comprehensive coverage. These include Regional Internet Registries (ARIN, RIPE, APNIC, etc.), which maintain official records of IP address allocations and ASN assignments.",
          "They also query routing databases that contain information about how networks are connected and advertised through BGP (Border Gateway Protocol). This routing information often reveals network relationships that aren't obvious from registry data alone.",
          "Additional sources include WHOIS databases, certificate transparency logs, DNS records, and various threat intelligence feeds that provide context about network usage and organizational relationships."
        ],
        keyPoints: [
          "Regional Internet Registries provide authoritative allocation data",
          "BGP routing databases reveal network relationships and advertisements",
          "WHOIS databases provide contact and organizational information",
          "Certificate transparency logs reveal domain and subdomain usage"
        ]
      },
      {
        title: "Complementary Capabilities and Use Cases",
        icon: "fa-puzzle-piece",
        content: [
          "Amass Intel and Metabigor complement each other by using different approaches and data sources. Amass Intel tends to be more comprehensive and methodical, while Metabigor is often faster and more focused on specific types of intelligence.",
          "Using both tools together provides better coverage because they may discover different aspects of an organization's infrastructure. Some networks might be found by one tool but not the other due to differences in data sources or query methodologies.",
          "The combination is particularly powerful for large organizations with complex infrastructure, acquisitions, or subsidiary relationships that might not be immediately obvious from a single data source."
        ],
        keyPoints: [
          "Different tools use different data sources and methodologies",
          "Combined use provides more comprehensive coverage",
          "Particularly effective for complex organizational structures",
          "Cross-validation helps confirm findings and reduce false positives"
        ]
      }
    ],
    practicalTips: [
      "Run both tools against the same target to maximize discovery",
      "Start with organization names, then drill down into specific ASNs or domains",
      "Pay attention to timing - some data sources update more frequently than others",
      "Cross-reference findings with manual WHOIS lookups for validation",
      "Look for patterns in discovered ranges that might indicate subnet organization",
      "Don't forget to check for IPv6 ranges in addition to IPv4",
      "Consider running tools from different geographic locations for different perspectives"
    ],
    furtherReading: [
      {
        title: "OWASP Amass Project",
        url: "https://owasp.org/www-project-amass/",
        description: "Official documentation for the Amass project"
      },
      {
        title: "Metabigor GitHub Repository",
        url: "https://github.com/j3ssie/metabigor",
        description: "Source code and documentation for Metabigor"
      },
      {
        title: "OSINT Framework",
        url: "https://osintframework.com/",
        description: "Comprehensive collection of OSINT tools and resources"
      },
      {
        title: "BGP Routing and Internet Infrastructure",
        url: "https://www.cloudflare.com/learning/security/glossary/what-is-bgp/",
        description: "Understanding how BGP routing works and why it matters for infrastructure discovery"
      }
    ]
  },

  liveWebServersMethodology: {
    title: "Network Infrastructure Discovery: From IP Ranges to Live Web Servers",
    overview: "This phase of the bug bounty methodology focuses on converting discovered network ranges into actionable targets by identifying live web services running on the organization's infrastructure. This bridges the gap between network reconnaissance and target identification.",
    sections: [
      {
        title: "Understanding Network Infrastructure Discovery",
        icon: "fa-network-wired",
        content: [
          "Network Infrastructure Discovery is a critical phase in the bug bounty methodology that comes after ASN and network range discovery. While the previous phase identified which IP ranges belong to the organization, this phase determines what's actually running on those IP addresses.",
          "This phase is essential because organizations often have web services, APIs, admin panels, and applications running on their internal infrastructure that aren't discoverable through domain-based reconnaissance. These services might include development environments, staging servers, admin interfaces, monitoring dashboards, or legacy applications.",
          "The goal is to systematically scan the discovered network ranges to identify live hosts and then determine which of those hosts are running web services that could be potential bug bounty targets. This process transforms abstract IP ranges into concrete, testable targets."
        ]
      },
      {
        title: "Methodology Position and Objectives",
        icon: "fa-bullseye",
        content: [
          "We're in the 'Network Infrastructure Discovery' phase, which sits between 'ASN/Network Range Discovery' and 'Target Selection/Vulnerability Assessment'. At this point, we have IP ranges but need to find actual services running on those ranges.",
          "Our primary objective is to discover live web servers, APIs, and other HTTP/HTTPS services running on IP addresses within the organization's network ranges. These services represent potential bug bounty targets that might not be discoverable through traditional domain enumeration.",
          "Secondary objectives include gathering initial metadata about discovered services (technologies, server headers, response characteristics) and identifying potentially high-value targets such as admin interfaces, development environments, or services with unusual configurations.",
          "The output of this phase should be a comprehensive list of live web servers with URLs, IP addresses, ports, and basic metadata that can be used for further vulnerability assessment and testing."
        ]
      },
      {
        title: "What We're Looking For",
        icon: "fa-search",
        content: [
          "**Administrative Interfaces**: Admin panels, configuration interfaces, and management consoles that might have weak authentication or expose sensitive functionality.",
          "**Development and Staging Environments**: Test servers, development environments, and staging applications that often have relaxed security controls and might contain debugging information or test data.",
          "**Legacy Applications**: Older web applications that might be running outdated software versions with known vulnerabilities or security misconfigurations.",
          "**Internal APIs and Services**: REST APIs, GraphQL endpoints, microservices, and other programmatic interfaces that might lack proper authentication or authorization controls.",
          "**Monitoring and Infrastructure Tools**: Dashboards, monitoring interfaces, CI/CD pipelines, and infrastructure management tools that might expose sensitive information about the organization's setup."
        ]
      },
      {
        title: "Strategic Value in Bug Bounty Hunting",
        icon: "fa-trophy",
        content: [
          "This phase often uncovers the highest-value targets in bug bounty programs because internal infrastructure services frequently have different security models than public-facing applications. Many organizations focus their security efforts on public websites while internal services may have weaker controls.",
          "Services discovered through this method are often not included in traditional security testing or penetration tests, making them more likely to contain undiscovered vulnerabilities. They may also have been forgotten or poorly maintained, leading to security issues.",
          "The systematic nature of this approach ensures comprehensive coverage of the organization's attack surface, reducing the likelihood of missing important targets that could lead to significant vulnerabilities.",
          "This methodology can reveal the organization's technology stack, internal architecture, and security practices, providing valuable context for further testing and helping prioritize the most promising targets."
        ]
      }
    ],
    practicalTips: [
      "Always ensure you have proper authorization before scanning any IP ranges, especially those that might belong to third parties - review your program scope carefully",
      "Use rate limiting (start with 10-50 requests/second) and respectful scanning practices to avoid overwhelming target infrastructure or triggering WAFs",
      "Pay special attention to non-standard ports like 8080, 8443, 3000, 4000, 5000, 9000 - these often host internal services, admin panels, or development environments",
      "Document the context and location of discovered services - internal services might have different security expectations and disclosure processes",
      "Look for services that return unusual status codes (403, 401, 500) or interesting headers (X-Powered-By, Server) that might indicate custom applications or misconfigurations",
      "Use tools like Shodan (shodan.io) and Censys (censys.io) to cross-reference your findings with known internet-wide scan data",
      "Consider using VPN services or scanning from different geographic locations if certain services appear to be geo-blocked"
    ],
    furtherReading: [
      {
        title: "OWASP Testing Guide - Infrastructure Security Testing",
        url: "https://owasp.org/www-project-web-security-testing-guide/v42/4-Web_Application_Security_Testing/02-Configuration_and_Deployment_Management_Testing/",
        description: "Comprehensive guide on testing network infrastructure and deployment configurations"
      },
      {
        title: "Shodan Search Engine",
        url: "https://www.shodan.io/",
        description: "Search engine for internet-connected devices - useful for cross-referencing discovered services"
      },
      {
        title: "Censys Internet Search",
        url: "https://censys.io/",
        description: "Internet-wide scanning platform for discovering and analyzing internet infrastructure"
      },
      {
        title: "Bug Bounty Methodology v4 by @jhaddix",
        url: "https://github.com/jhaddix/tbhm",
        description: "Comprehensive bug bounty methodology including network reconnaissance techniques"
      },
      {
        title: "Internal Network Penetration Testing",
        url: "https://book.hacktricks.xyz/generic-methodologies-and-resources/pentesting-network",
        description: "HackTricks guide on network penetration testing methodologies and techniques"
      }
    ]
  },

  ipPortScanningProcess: {
    title: "IP/Port Scanning: Technical Deep Dive into Live Service Discovery",
    overview: "Understanding the technical process of how IP/Port scanning systematically converts network ranges into live web servers, including the two-phase approach of host discovery followed by service enumeration.",
    sections: [
      {
        title: "Two-Phase Scanning Methodology",
        icon: "fa-layer-group",
        content: [
          "The IP/Port scanning process uses a two-phase approach to efficiently discover live web servers. Phase 1 focuses on host discovery - identifying which IP addresses in the network ranges are actually alive and responding. Phase 2 focuses on service enumeration - determining which live hosts are running web services.",
          "This two-phase approach is much more efficient than trying to scan all possible web ports on all possible IP addresses. By first identifying live hosts, we can focus our more intensive service scanning on targets that are actually responsive.",
          "Phase 1 uses TCP connect probes on common service ports (80, 443, 22, 21, 25, 53, etc.) to quickly determine host liveness. If any port responds, the host is marked as live. Phase 2 then performs detailed scanning of web-specific ports only on the live hosts.",
          "The process is designed to be respectful and efficient, using concurrency controls, timeouts, and rate limiting to avoid overwhelming target infrastructure while still providing comprehensive coverage."
        ]
      },
      {
        title: "Phase 1: Host Discovery Process",
        icon: "fa-search-location",
        content: [
          "Host discovery begins by parsing the consolidated network ranges (CIDR blocks) and generating all possible IP addresses within those ranges. For large networks, the system limits scanning to prevent memory issues and ensure reasonable scan times.",
          "Each IP address is probed using TCP connect attempts on a carefully selected list of common service ports: 80 (HTTP), 443 (HTTPS), 22 (SSH), 21 (FTP), 25 (SMTP), 53 (DNS), 110 (POP3), 995 (POP3S), 993 (IMAPS), and 143 (IMAP).",
          "The system uses concurrent goroutines with semaphore-based rate limiting to control the number of simultaneous connection attempts. Each probe has a 1-second timeout to quickly identify responsive hosts without waiting too long for unresponsive ones.",
          "When any port on a host responds, the IP address is marked as live and stored in the database along with the network range it belongs to. This creates an inventory of responsive hosts for the next phase."
        ]
      },
      {
        title: "Phase 2: Web Service Discovery",
        icon: "fa-globe",
        content: [
          "Once live hosts are identified, the system performs targeted port scanning specifically for web services. It scans a comprehensive list of web-related ports: 80, 443, 8080, 8443, 8000, 8001, 3000, 3001, 4000, 4001, 5000, 5001, 7000, 7001, 9000, 9001, and many others.",
          "For each open port discovered, the system attempts both HTTP and HTTPS connections to determine if a web service is running. It uses a custom HTTP client with TLS verification disabled to handle self-signed certificates and development environments.",
          "When a web service responds, the system extracts comprehensive metadata including HTTP status code, response time, server headers, page title, content length, and attempts to identify technologies based on headers and response characteristics.",
          "All discovered web servers are stored with their complete metadata, creating a comprehensive inventory of live web services across the organization's network infrastructure."
        ]
      },
      {
        title: "Technical Implementation Details",
        icon: "fa-cogs",
        content: [
          "**Concurrency Control**: The system uses semaphores to limit concurrent operations - typically 50 concurrent IP probes and 20 concurrent port scans per IP to balance speed with resource usage and target respect.",
          "**Rate Limiting**: Built-in delays and connection limits ensure the scanning doesn't overwhelm target infrastructure or trigger security monitoring systems.",
          "**Timeout Management**: Each phase uses appropriate timeouts - 1 second for host discovery probes, 1 second for port scans, and 5 seconds for HTTP requests to gather metadata.",
          "**Error Handling**: The system gracefully handles network errors, timeouts, and connection refused responses, ensuring that scanning continues even when individual probes fail.",
          "**Database Integration**: All results are stored in real-time, allowing for progress monitoring and ensuring that partial results are preserved even if scanning is interrupted."
        ]
      },
      {
        title: "Output and Results Processing",
        icon: "fa-list-alt",
        content: [
          "The scanning process produces several types of valuable output: discovered live IP addresses with their associated network ranges, live web servers with complete URLs and metadata, and scan statistics including total hosts scanned, live hosts found, and web services discovered.",
          "Each discovered web server includes the full URL, IP address, port, protocol (HTTP/HTTPS), HTTP status code, page title, server header information, detected technologies, response time, and content length.",
          "The results are automatically integrated into the framework's attack surface consolidation process, making discovered services available for further analysis, vulnerability assessment, and potential addition to scope targets.",
          "Scan progress is tracked in real-time, providing visibility into the number of network ranges processed, IP addresses scanned, live hosts discovered, and web services found."
        ]
      }
    ],
    practicalTips: [
      "Monitor scan progress through the results interface to understand the scope and effectiveness of the scan - large networks can take hours to complete",
      "Pay attention to non-standard ports (8080, 8443, 3000, 4000, 5000, 9000, 10000) - services on unusual ports often represent internal or development systems",
      "Look for patterns in discovered services that might indicate specific technologies (multiple Tomcat servers, Jenkins instances, etc.) or similar architectures",
      "Use the metadata to prioritize targets - look for interesting server headers (X-Powered-By: PHP/7.2.34), unusual status codes (403, 401), or revealing titles ('Admin Panel', 'Jenkins', 'Grafana')",
      "Consider the response times and service characteristics when identifying potentially high-value targets - slow responses might indicate complex applications",
      "Use tools like masscan or RustScan for initial port discovery if you need faster scanning, then use the framework for web service identification",
      "Cross-reference discovered services with CVE databases and exploit-db.com to identify known vulnerabilities in specific versions"
    ],
    furtherReading: [
      {
        title: "Nmap Port Scanning Techniques",
        url: "https://nmap.org/book/man-port-scanning-techniques.html",
        description: "Comprehensive guide to port scanning techniques and methodologies"
      },
      {
        title: "masscan - Fast Port Scanner",
        url: "https://github.com/robertdavidgraham/masscan",
        description: "High-speed port scanner capable of scanning the entire internet quickly"
      },
      {
        title: "RustScan - Modern Port Scanner",
        url: "https://github.com/RustScan/RustScan",
        description: "Fast, modern port scanner with scripting capabilities"
      },
      {
        title: "Common Ports List - SpeedGuide",
        url: "https://www.speedguide.net/ports.php",
        description: "Comprehensive database of TCP and UDP port assignments"
      },
      {
        title: "Web Application Firewalls (WAF) Bypass Techniques",
        url: "https://github.com/0xInfection/Awesome-WAF",
        description: "Collection of WAF bypass techniques and tools for when scanning triggers security measures"
      },
      {
        title: "TCP Connect Scan Deep Dive",
        url: "https://nmap.org/book/scan-methods-connect-scan.html",
        description: "Technical details about TCP connect scanning methodology and advantages"
      }
    ]
  },

  liveWebServerTools: {
    title: "Tools and Techniques for Live Web Server Discovery and Analysis",
    overview: "A comprehensive guide to the tools, techniques, and technologies used in the live web server discovery process, including custom scanning tools, metadata gathering, and analysis techniques.",
    sections: [
      {
        title: "Custom IP/Port Scanning Engine",
        icon: "fa-tools",
        content: [
          "The framework uses a custom-built IP/Port scanning engine specifically designed for bug bounty reconnaissance. Unlike general-purpose network scanners, this engine is optimized for discovering web services across large network ranges while maintaining respectful scanning practices.",
          "The engine is implemented in Go for high performance and efficient concurrency handling. It uses native TCP connect scans rather than SYN scans, which are more reliable across different network configurations and don't require special privileges.",
          "Key features include automatic CIDR parsing and IP generation, intelligent rate limiting based on network conditions, comprehensive port coverage for web services, real-time progress tracking and result storage, and graceful error handling for network issues.",
          "The scanning engine integrates directly with the framework's database, storing results in real-time and providing immediate feedback on discovery progress. This allows for interruption and resumption of large scans without losing progress."
        ]
      },
      {
        title: "Host Discovery Techniques",
        icon: "fa-broadcast-tower",
        content: [
          "Host discovery uses TCP connect probes on a carefully curated list of common service ports. This approach is more reliable than ICMP ping, which is often blocked by firewalls, and provides immediate insight into what services might be running.",
          "The port selection includes both well-known ports (80, 443, 22) and common alternative ports (8080, 8443, 3000) to maximize the chances of detecting live hosts across different environments and configurations.",
          "The system uses a timeout-based approach where each port probe has a 1-second timeout. If any port responds within the timeout, the host is marked as live. This balances speed with thoroughness.",
          "Concurrent probing with semaphore-based rate limiting ensures efficient scanning while preventing network congestion or triggering security monitoring systems that might block further scanning attempts."
        ]
      },
      {
        title: "Web Service Enumeration",
        icon: "fa-globe-americas",
        content: [
          "Once live hosts are identified, the system performs targeted web service enumeration using an extensive list of web-related ports. This includes standard ports (80, 443), common alternatives (8080, 8443), development ports (3000, 3001, 4000, 4001), and less common but frequently used ports.",
          "For each open port, the system attempts both HTTP and HTTPS connections to account for services that might be running SSL/TLS on non-standard ports. The HTTP client is configured with disabled certificate verification to handle self-signed certificates common in internal environments.",
          "The enumeration process extracts comprehensive metadata from each discovered service: HTTP status codes, response headers (especially Server headers), page titles, content lengths, response times, and basic technology detection based on headers and response characteristics.",
          "All discovered web services are stored with their complete metadata, creating a rich inventory that can be used for prioritization and further analysis."
        ]
      },
      {
        title: "Metadata Gathering with Katana",
        icon: "fa-spider",
        content: [
          "After initial web service discovery, the framework uses Katana (a next-generation crawling and spidering framework) to gather additional metadata and context about discovered services. This provides deeper insight than basic HTTP probes.",
          "Katana performs intelligent crawling of discovered web services, following links, analyzing JavaScript, and mapping out the application structure. This can reveal additional endpoints, API paths, and functionality that might not be apparent from the initial discovery.",
          "The crawling process is configured with appropriate rate limits and depth restrictions to avoid overwhelming target services while still gathering comprehensive information about the application structure and content.",
          "Results from Katana include discovered URLs, page content analysis, technology stack identification, and potential security issues like exposed configuration files or sensitive information in page content."
        ]
      },
      {
        title: "Technology and Framework Detection",
        icon: "fa-microchip",
        content: [
          "The framework includes sophisticated technology detection capabilities that analyze HTTP headers, response content, and other indicators to identify the underlying technologies and frameworks powering discovered web services.",
          "Technology detection examines Server headers, X-Powered-By headers, Set-Cookie headers for framework-specific patterns, Content-Type headers, and response body content for technology-specific signatures and patterns.",
          "Common technologies that can be identified include web servers (Apache, Nginx, IIS), programming languages and frameworks (PHP, ASP.NET, Node.js, Python), content management systems (WordPress, Drupal, Joomla), and application frameworks (Spring, Laravel, Django).",
          "This information is crucial for prioritizing targets and understanding potential attack vectors, as different technologies have different common vulnerabilities and security considerations."
        ]
      },
      {
        title: "Results Analysis and Prioritization",
        icon: "fa-chart-line",
        content: [
          "The framework provides comprehensive results analysis tools that help prioritize discovered web services based on various factors including technology stack, response characteristics, and potential security impact.",
          "Key indicators for high-priority targets include non-standard ports (often internal services), unusual or interesting page titles, missing or unusual security headers, error messages or debug information, and services running on development-related ports.",
          "The system automatically flags potential high-value targets such as admin interfaces, development environments, API endpoints, monitoring dashboards, and services with unusual configurations or exposed sensitive information.",
          "Results can be filtered and sorted by various criteria including IP address, port, status code, title content, server header, and detected technologies to help focus testing efforts on the most promising targets."
        ]
      }
    ],
    practicalTips: [
      "Use the port and technology information to understand the target's infrastructure and prioritize testing efforts - look for technology clusters or patterns",
      "Pay special attention to services running on non-standard ports (8080, 8443, 3000, 4000, 5000, 9000) - these often represent internal or development systems with weaker security",
      "Look for patterns in server headers and technologies that might indicate a specific technology stack (LAMP, MEAN, .NET) or configuration management system",
      "Services with interesting titles ('Admin', 'Dashboard', 'Jenkins', 'Grafana', 'phpMyAdmin') or unusual status codes (401, 403, 500) often warrant immediate investigation",
      "Use the response time information to understand service performance and potential hosting locations - consistent fast responses might indicate CDN usage",
      "Leverage Wappalyzer browser extension or online tools to cross-reference technology detection with manual verification",
      "Document discovered admin interfaces, development tools, and monitoring systems as these often have default credentials or known vulnerabilities"
    ],
    furtherReading: [
      {
        title: "Katana - Next-generation Crawling Framework",
        url: "https://github.com/projectdiscovery/katana",
        description: "Modern web crawling and spidering framework by ProjectDiscovery for comprehensive asset discovery"
      },
      {
        title: "httpx - Fast HTTP Toolkit",
        url: "https://github.com/projectdiscovery/httpx",
        description: "Fast and multi-purpose HTTP toolkit for running multiple web probes"
      },
      {
        title: "Wappalyzer Technology Detection",
        url: "https://www.wappalyzer.com/",
        description: "Technology profiler that identifies technologies used on websites"
      },
      {
        title: "CVE Database - MITRE",
        url: "https://cve.mitre.org/",
        description: "Official CVE database for looking up known vulnerabilities in discovered technologies"
      },
      {
        title: "Exploit Database",
        url: "https://www.exploit-db.com/",
        description: "Archive of public exploits and corresponding vulnerable software"
      },
      {
        title: "Default Credentials Cheat Sheet",
        url: "https://github.com/ihebski/DefaultCreds-cheat-sheet",
        description: "Comprehensive list of default credentials for various systems and applications"
      },
      {
        title: "HackerOne Bug Bounty Methodology",
        url: "https://www.hackerone.com/ethical-hacker/methodology",
        description: "Official bug bounty methodology guide covering reconnaissance and target prioritization"
      },
      {
        title: "OWASP Web Security Testing Guide",
        url: "https://owasp.org/www-project-web-security-testing-guide/",
        description: "Comprehensive guide for testing web application security across different technologies"
      }
    ]
  },

  rootDomainMethodology: {
    title: "Root Domain Discovery: Expanding Organizational Attack Surface",
    overview: "Root Domain Discovery is a critical reconnaissance phase that systematically identifies all domains owned or controlled by the target organization, expanding the attack surface beyond any single domain to reveal the complete digital footprint of the company.",
    sections: [
      {
        title: "Understanding Root Domain Discovery in Bug Bounty Methodology",
        icon: "fa-sitemap",
        content: [
          "Root Domain Discovery sits early in the reconnaissance phase, typically after initial target identification but before deep subdomain enumeration. Unlike subdomain discovery which finds variations of a known domain, root domain discovery identifies entirely separate domains owned by the organization.",
          "This phase is essential because modern organizations rarely operate under a single domain. They often own multiple domains for different business units, geographical regions, subsidiary companies, acquisitions, legacy brands, and specialized business functions.",
          "The methodology recognizes that many high-impact vulnerabilities are found on 'forgotten' or less-monitored domains that don't receive the same security attention as primary corporate websites. These domains often represent legacy systems, development environments, or acquired assets with weaker security controls.",
          "Root domain discovery provides the foundation for comprehensive reconnaissance by ensuring we don't miss any significant parts of the organization's digital infrastructure that could contain valuable targets."
        ]
      },
      {
        title: "Organizational Digital Infrastructure Patterns",
        icon: "fa-building",
        content: [
          "Large organizations typically have complex domain portfolios reflecting their business structure. Primary business domains handle main corporate functions, while subsidiary domains serve acquired companies or business units that maintain separate digital identities.",
          "Geographical domains are common for multinational companies operating in different regions, often using country-specific top-level domains or regional naming conventions. Development and staging domains support software development lifecycle with names like 'dev-', 'staging-', or 'test-' prefixes.",
          "Legacy domains remain from previous branding, marketing campaigns, or business initiatives that may no longer be actively maintained but still contain functional systems. Acquisition domains come from companies that were purchased but maintain separate digital infrastructure.",
          "Specialized function domains serve specific business needs like customer support, partner portals, vendor management, or industry-specific services that require separate branding or technical infrastructure."
        ]
      },
      {
        title: "Strategic Value of Root Domain Discovery",
        icon: "fa-crosshairs",
        content: [
          "Root domain discovery often reveals the highest-value targets in bug bounty programs because secondary domains frequently receive less security attention than primary corporate websites. Security teams may focus resources on main business domains while neglecting subsidiary or legacy domains.",
          "Forgotten or legacy domains are particularly valuable because they may run outdated software, lack modern security controls, or have been excluded from regular security assessments. These domains often contain the same sensitive data or functionality as primary domains but with weaker protections.",
          "Subsidiary and acquisition domains may have different security standards, older technologies, or integration points with main corporate systems that create unique attack vectors. They might also have different bug bounty scope rules or disclosure processes.",
          "The comprehensive nature of root domain discovery ensures systematic coverage of the organization's attack surface, reducing the likelihood of missing critical assets that could lead to significant security findings."
        ]
      },
      {
        title: "Methodology Positioning and Workflow Integration",
        icon: "fa-project-diagram",
        content: [
          "Root Domain Discovery occurs after initial target identification but before intensive subdomain enumeration, creating a complete list of domains that will feed into subsequent reconnaissance phases. This positioning maximizes efficiency by ensuring all organizational domains are identified before deep-dive analysis.",
          "The phase integrates with both manual research (company information gathering) and automated tools (no-API and API-based discovery), providing multiple discovery vectors to ensure comprehensive coverage of the organization's domain portfolio.",
          "Results from this phase directly feed into scope target creation, subdomain enumeration, and network range discovery, making it a critical foundation for all subsequent testing activities. The quality of root domain discovery often determines the overall success of the engagement.",
          "The methodology emphasizes both quantity (finding all domains) and quality (validating domain ownership and relevance) to ensure that discovered domains are legitimate organizational assets rather than false positives or unrelated domains."
        ]
      }
    ],
    practicalTips: [
      "Start with basic company research to understand the organizational structure, subsidiaries, and business units before running automated tools",
      "Use multiple discovery methods (Google Dorking, CRT, Reverse WHOIS) as each technique may find domains that others miss due to different data sources",
      "Pay attention to domain naming patterns and conventions used by the organization - this can help identify additional domains manually",
      "Validate domain ownership by checking WHOIS records, website content, and SSL certificate information to ensure domains actually belong to the target organization",
      "Look for seasonal or campaign-specific domains that might be temporarily inactive but still contain interesting infrastructure",
      "Consider international and regional variations of company names when searching, especially for multinational organizations",
      "Document the discovery method for each domain to understand data reliability and help with validation decisions"
    ],
    furtherReading: [
      {
        title: "OWASP Testing Guide - Information Gathering",
        url: "https://owasp.org/www-project-web-security-testing-guide/v42/4-Web_Application_Security_Testing/01-Information_Gathering/",
        description: "Comprehensive guide to gathering information about target organizations and their digital assets"
      },
      {
        title: "Domain Research and OSINT Techniques",
        url: "https://osintframework.com/",
        description: "Collection of open source intelligence tools and techniques for domain and organizational research"
      },
      {
        title: "Bug Bounty Reconnaissance Methodology",
        url: "https://github.com/jhaddix/tbhm",
        description: "The Bug Hunters Methodology covering reconnaissance techniques including domain discovery"
      },
      {
        title: "Corporate Structure Research Guide",
        url: "https://www.sec.gov/edgar.shtml",
        description: "SEC EDGAR database for researching corporate structures, subsidiaries, and business relationships"
      }
    ]
  },

  noApiKeyTools: {
    title: "Google Dorking, Certificate Transparency, and Reverse WHOIS: OSINT Domain Discovery",
    overview: "These three complementary OSINT techniques provide comprehensive root domain discovery without requiring premium API access, leveraging publicly available data sources to identify organizational domains through different discovery vectors.",
    sections: [
      {
        title: "Google Dorking: Search Engine Intelligence",
        icon: "fa-search",
        content: [
          "Google Dorking (also called Google Hacking) uses advanced search operators to query search engines for specific information about target organizations. For domain discovery, it leverages the vast amount of indexed content that mentions organizational domains in various contexts.",
          "The technique works by using search operators like 'site:', 'inurl:', 'intitle:', and 'intext:' combined with organizational names, known domains, and related keywords to find mentions of domains in public documents, job postings, news articles, and other indexed content.",
          "Google Dorking is particularly effective at finding domains mentioned in corporate communications, press releases, job descriptions, conference presentations, and other business documents that might reference internal or subsidiary domains not found through other methods.",
          "The approach can discover domains used for specific business functions, geographic regions, or temporary campaigns that might not be obvious from automated scanning but are referenced in public content."
        ],
        keyPoints: [
          "Uses search engine operators to find domain mentions in indexed content",
          "Effective for finding domains mentioned in corporate documents and communications",
          "Can discover context about domain purposes and business functions",
          "Completely free and doesn't require any API keys or premium access"
        ],
        examples: [
          {
            code: 'site:*.example.com -site:www.example.com',
            description: "Find subdomains of example.com excluding the main www domain"
          },
          {
            code: '"Example Corporation" site:linkedin.com',
            description: "Find LinkedIn profiles mentioning the organization"
          },
          {
            code: 'inurl:example OR intext:"example.com" -site:example.com',
            description: "Find pages mentioning the domain but not hosted on it"
          }
        ]
      },
      {
        title: "Certificate Transparency (CRT): SSL Certificate Intelligence",
        icon: "fa-certificate",
        content: [
          "Certificate Transparency is a public logging system that records all SSL/TLS certificates issued by Certificate Authorities. This creates a searchable database of all domains that have obtained SSL certificates, including internal and non-public domains.",
          "The system was created to detect fraudulent certificates, but it provides invaluable intelligence for security researchers. Organizations often obtain certificates for internal domains, development environments, and subsidiary domains that aren't publicly advertised but are discoverable through CT logs.",
          "CT logs are particularly valuable because they capture domains at the time certificates are issued, providing historical data about an organization's infrastructure evolution. They often reveal domains that may no longer be active but still contain interesting infrastructure.",
          "The technique is highly reliable because certificate issuance is an authoritative action - if a domain appears in CT logs, someone with control over that domain requested a certificate for it, indicating organizational ownership or control."
        ],
        keyPoints: [
          "Searches public logs of all SSL/TLS certificates issued for domains",
          "Reveals internal and non-public domains that organizations secure with SSL",
          "Provides historical data about organizational domain usage over time",
          "Highly reliable data source due to the authoritative nature of certificate issuance"
        ],
        examples: [
          {
            code: "crt.sh query: %.example.com",
            description: "Find all certificates issued for subdomains of example.com"
          },
          {
            code: "CT search: organization name",
            description: "Search for certificates issued to the organization by name"
          },
          {
            code: "Historical CT data: 2020-2024",
            description: "Review certificate history to find discontinued or changed domains"
          }
        ]
      },
      {
        title: "Reverse WHOIS: Registration Intelligence",
        icon: "fa-address-card",
        content: [
          "Reverse WHOIS queries domain registration databases using organizational information like company names, email addresses, phone numbers, or physical addresses to find all domains registered with that information. This reveals domains that share common registration details.",
          "The technique is particularly effective for finding subsidiary domains, acquisition-related domains, and legacy domains that were registered using the same contact information or organizational details, even if they're not obviously related to the main business.",
          "Reverse WHOIS can reveal historical relationships between domains, showing how an organization's domain portfolio has evolved through acquisitions, business changes, or administrative updates. It often finds domains that other techniques miss.",
          "The approach is especially valuable for large organizations with complex corporate structures, as it can reveal domains owned by subsidiaries, holding companies, or business units that might not be obvious from company research alone."
        ],
        keyPoints: [
          "Searches domain registration records using organizational contact information",
          "Effective for finding subsidiary, acquisition, and legacy domains",
          "Reveals historical relationships and organizational evolution",
          "Can discover domains owned by related business entities"
        ],
        examples: [
          {
            code: "Reverse WHOIS: 'Example Corporation'",
            description: "Find domains registered to the organization by name"
          },
          {
            code: "Email search: admin@example.com",
            description: "Find domains registered using organizational email addresses"
          },
          {
            code: "Phone search: +1-555-0100",
            description: "Find domains registered using organizational phone numbers"
          }
        ]
      },
      {
        title: "Complementary Capabilities and Data Coverage",
        icon: "fa-layer-group",
        content: [
          "These three techniques provide complementary coverage because they access different data sources and discovery vectors. Google Dorking finds domains mentioned in public content, CRT finds domains with SSL certificates, and Reverse WHOIS finds domains with shared registration information.",
          "Each technique has different strengths and may discover domains that others miss. Google Dorking excels at finding contextual information about domain purposes, CRT is excellent for comprehensive subdomain discovery, and Reverse WHOIS is unmatched for finding related organizational domains.",
          "The combination provides temporal coverage spanning current active domains (Google Dorking), recent certificate activity (CRT), and historical registration relationships (Reverse WHOIS), ensuring comprehensive discovery across different time periods.",
          "Using all three techniques together provides validation and cross-referencing opportunities - domains found by multiple methods are more likely to be legitimate organizational assets, while unique findings from each method expand the total discovery scope."
        ],
        keyPoints: [
          "Each technique accesses different data sources and provides unique discovery capabilities",
          "Combined use provides comprehensive temporal and methodological coverage",
          "Cross-referencing results helps validate organizational ownership",
          "Complementary strengths ensure no single discovery vector is missed"
        ]
      }
    ],
    practicalTips: [
      "Start with Google Dorking using known organizational information and domain patterns to understand naming conventions and business structure",
      "Use specific search operators like 'site:' and 'inurl:' to find domain mentions in job postings, press releases, and corporate communications",
      "Search Certificate Transparency logs using both exact domain matches and wildcard searches to find all certificate-protected domains",
      "Try multiple variations of organization names in Reverse WHOIS searches, including abbreviations, legal entity names, and historical company names",
      "Cross-reference findings between techniques - domains found by multiple methods are more likely to be legitimate organizational assets",
      "Pay attention to domain registration dates and certificate issuance dates to understand organizational changes and acquisitions",
      "Use domain validation techniques like WHOIS lookups and website content analysis to confirm organizational ownership of discovered domains"
    ],
    furtherReading: [
      {
        title: "Google Search Operators Guide",
        url: "https://support.google.com/websearch/answer/2466433",
        description: "Official Google documentation for advanced search operators and techniques"
      },
      {
        title: "Certificate Transparency - crt.sh",
        url: "https://crt.sh/",
        description: "Popular certificate transparency log search interface for domain discovery"
      },
      {
        title: "WHOIS Database Search",
        url: "https://whois.net/",
        description: "Domain registration information lookup and reverse WHOIS search capabilities"
      },
      {
        title: "DomainTools Reverse WHOIS",
        url: "https://reversewhois.domaintools.com/",
        description: "Professional reverse WHOIS search tool for finding related domains"
      },
      {
        title: "Google Hacking Database (GHDB)",
        url: "https://www.exploit-db.com/google-hacking-database",
        description: "Collection of Google search operators for security research and information gathering"
      },
      {
        title: "OSINT Framework - Domain Research",
        url: "https://osintframework.com/",
        description: "Comprehensive collection of domain research and OSINT tools"
      }
    ]
  },

  rootDomainPrioritization: {
    title: "Root Domain Prioritization: Strategic Target Selection and Analysis",
    overview: "Effective root domain prioritization helps bug bounty hunters focus their limited time and resources on the most promising targets by analyzing domain characteristics, business context, and potential security implications.",
    sections: [
      {
        title: "High-Value Domain Characteristics",
        icon: "fa-bullseye",
        content: [
          "Forgotten or legacy domains represent some of the highest-value targets because they often run outdated software, lack modern security controls, or have been excluded from regular security assessments while still containing sensitive functionality or data.",
          "Subsidiary and acquisition domains frequently have different security standards, older technologies, or integration points with main corporate systems. They may operate under different bug bounty programs or have unique disclosure requirements.",
          "Development and staging domains often have relaxed security controls, debugging features enabled, or test data that provides insights into production systems. They may also lack the monitoring and incident response capabilities of production environments.",
          "Geographic and regional domains may serve different regulatory environments, use different technology stacks, or have varying security requirements based on local compliance needs and business practices."
        ]
      },
      {
        title: "Domain Naming Pattern Analysis",
        icon: "fa-code",
        content: [
          "Technical naming patterns like 'dev-', 'staging-', 'test-', 'admin-', 'internal-', or 'beta-' often indicate development environments, administrative interfaces, or internal tools that may have weaker security controls than production systems.",
          "Geographic indicators in domain names (country codes, city names, regional abbreviations) can reveal international operations, localized services, or regional business units that might have different security postures or regulatory requirements.",
          "Business function indicators like 'support-', 'partner-', 'vendor-', 'api-', or 'mobile-' suggest specialized services that might have unique authentication mechanisms, integration points, or data handling practices.",
          "Temporal or campaign-specific naming patterns (years, product launches, marketing campaigns) often indicate domains that were created for specific initiatives and may have been deprioritized or forgotten over time."
        ]
      },
      {
        title: "Business Context and Risk Assessment",
        icon: "fa-chart-line",
        content: [
          "Understanding the business purpose of discovered domains helps prioritize targets based on potential impact. Customer-facing domains might contain personal data, while partner portals might provide access to business systems or supply chain infrastructure.",
          "Domain age and last-seen activity provide insights into maintenance levels and security attention. Recently active domains are more likely to be monitored, while dormant domains might have been forgotten but still contain exploitable services.",
          "Technology stack analysis through initial reconnaissance (HTTP headers, certificate information, error pages) can reveal outdated software versions, misconfigurations, or technologies with known vulnerabilities.",
          "Integration points and business relationships suggested by domain purposes can indicate potential pivot opportunities or access to broader organizational infrastructure through compromised subsidiary or partner systems."
        ]
      },
      {
        title: "Security Posture Indicators",
        icon: "fa-shield-alt",
        content: [
          "SSL certificate information provides insights into domain maintenance and security practices. Expired certificates, self-signed certificates, or certificates with unusual issuers might indicate less-maintained infrastructure.",
          "DNS configuration analysis can reveal misconfigurations, outdated records, or unusual hosting arrangements that might indicate security weaknesses or forgotten infrastructure components.",
          "HTTP security headers and response characteristics provide immediate insights into security posture. Missing security headers, verbose error messages, or unusual server signatures might indicate weaker security controls.",
          "Website content and functionality analysis helps understand the domain's current state, business purpose, and potential attack surface. Login pages, administrative interfaces, or API endpoints suggest areas for deeper investigation."
        ]
      },
      {
        title: "Prioritization Framework and Decision Making",
        icon: "fa-sort-amount-down",
        content: [
          "High-priority domains typically include administrative interfaces, development environments, forgotten legacy systems, and subsidiary domains with potential integration points to main corporate infrastructure.",
          "Medium-priority domains might include regional business sites, marketing campaign domains, or specialized business function domains that could contain sensitive data but may have standard security controls.",
          "Lower-priority domains often include purely informational sites, redirect domains, or well-maintained subsidiary domains that appear to have modern security controls and regular maintenance.",
          "The prioritization framework should consider both potential impact (what could be achieved through compromise) and likelihood of success (based on observed security indicators and maintenance levels)."
        ]
      }
    ],
    practicalTips: [
      "Research discovered domains through business context - understanding their purpose helps assess potential impact and security expectations",
      "Look for domains with technical naming patterns (dev-, admin-, test-) as these often indicate development or administrative environments with relaxed security",
      "Check SSL certificate information for each domain - expired or self-signed certificates often indicate less-maintained infrastructure",
      "Analyze HTTP responses for security headers, server information, and error messages that might indicate security posture",
      "Pay attention to domain registration dates relative to acquisition announcements or business changes - recently acquired domains might have integration vulnerabilities",
      "Consider the geographical and regulatory context of international domains - different regions may have varying security standards",
      "Use tools like Wappalyzer or BuiltWith to analyze technology stacks and identify potentially vulnerable or outdated components",
      "Document findings and prioritization rationale to help with future target selection and time management decisions"
    ],
    furtherReading: [
      {
        title: "OWASP Top 10 - Security Risks",
        url: "https://owasp.org/www-project-top-ten/",
        description: "Understanding common web application security risks to help assess domain vulnerability potential"
      },
      {
        title: "Wappalyzer Technology Detection",
        url: "https://www.wappalyzer.com/",
        description: "Tool for analyzing website technology stacks and identifying potentially vulnerable components"
      },
      {
        title: "SSL Labs Server Test",
        url: "https://www.ssllabs.com/ssltest/",
        description: "Comprehensive SSL/TLS configuration analysis for assessing domain security posture"
      },
      {
        title: "SecurityHeaders.com",
        url: "https://securityheaders.com/",
        description: "Tool for analyzing HTTP security headers and identifying missing security controls"
      },
      {
        title: "Corporate Information Research",
        url: "https://www.sec.gov/edgar.shtml",
        description: "SEC EDGAR database for researching corporate structures, acquisitions, and business relationships"
      },
      {
        title: "Bug Bounty Methodology - Target Prioritization",
        url: "https://github.com/jhaddix/tbhm",
        description: "Comprehensive methodology guide including target selection and prioritization strategies"
      }
    ]
  }
}; 