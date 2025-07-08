import React, { useState, useEffect, useMemo } from 'react';
import { Modal, Table, Badge, Alert, Tabs, Tab, Card, Row, Col, Accordion, Button, Spinner } from 'react-bootstrap';

const KatanaCompanyResultsModal = ({ show, handleClose, activeTarget, mostRecentKatanaCompanyScan }) => {
  const [cloudAssets, setCloudAssets] = useState([]);
  const [cloudFindings, setCloudFindings] = useState([]);
  const [rawResults, setRawResults] = useState([]);
  const [allAvailableDomains, setAllAvailableDomains] = useState([]);
  const [baseDomains, setBaseDomains] = useState([]);
  const [wildcardDomains, setWildcardDomains] = useState([]);
  const [liveWebServers, setLiveWebServers] = useState([]);
  const [loadedRawResults, setLoadedRawResults] = useState({});
  const [loadingDomains, setLoadingDomains] = useState({});
  const [copyingDomains, setCopyingDomains] = useState({});
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const [activeTab, setActiveTab] = useState('assets');
  const [lastLoadedScanId, setLastLoadedScanId] = useState(null);

  // Combine all available domains like the config modal
  const combinedDomains = useMemo(() => {
    const combined = [];
    
    // Add consolidated company domains
    baseDomains.forEach(domain => {
      combined.push({
        domain,
        type: 'root',
        source: 'Company Domains',
        isWildcardTarget: wildcardDomains.some(wd => wd.rootDomain === domain)
      });
    });
    
    // Add wildcard discovered domains
    wildcardDomains.forEach(wd => {
      wd.discoveredDomains.forEach(discoveredDomain => {
        if (!combined.some(item => item.domain === discoveredDomain)) {
          combined.push({
            domain: discoveredDomain,
            type: 'wildcard',
            source: 'Wildcard Results',
            rootDomain: wd.wildcardTarget || wd.rootDomain
          });
        }
      });
    });
    
    // Add live web servers
    liveWebServers.forEach(server => {
      const domain = server.replace(/^https?:\/\//, '').replace(/\/.*$/, '');
      if (!combined.some(item => item.domain === domain)) {
        combined.push({
          domain: domain,
          type: 'live',
          source: 'Live Web Servers',
          url: server
        });
      }
    });
    
    return combined.sort((a, b) => a.domain.localeCompare(b.domain));
  }, [baseDomains, wildcardDomains, liveWebServers]);

  useEffect(() => {
    // Only load results when modal opens OR when we get a different scan_id
    if (show && mostRecentKatanaCompanyScan?.scan_id && 
        mostRecentKatanaCompanyScan.scan_id !== lastLoadedScanId) {
      loadResults();
      setLastLoadedScanId(mostRecentKatanaCompanyScan.scan_id);
    }
    
    // Load domains when modal opens
    if (show && activeTarget?.id) {
      loadAllAvailableDomains();
    }
  }, [show, mostRecentKatanaCompanyScan?.scan_id, activeTarget?.id]);

  // Update allAvailableDomains when combinedDomains changes
  useEffect(() => {
    setAllAvailableDomains(combinedDomains);
  }, [combinedDomains]);

  // Load wildcard domains and live web servers when baseDomains changes
  useEffect(() => {
    if (baseDomains.length > 0) {
      fetchWildcardDomains();
      fetchLiveWebServers();
    }
  }, [baseDomains]);

  const loadResults = async () => {
    if (!activeTarget?.id) return;

    setIsLoading(true);
    setError('');
    
    try {
      const [assetsResponse, findingsResponse, rawResponse] = await Promise.all([
        fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/katana-company/target/${activeTarget.id}/cloud-assets`
        ),
        fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/katana-company/target/${activeTarget.id}/cloud-findings`
        ),
        fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/katana-company/target/${activeTarget.id}/raw-results`
        )
      ]);

      if (!assetsResponse.ok || !findingsResponse.ok || !rawResponse.ok) {
        throw new Error('Failed to fetch cloud assets, findings, and raw results');
      }

      const assets = await assetsResponse.json();
      const findings = await findingsResponse.json();
      const raw = await rawResponse.json();

      setCloudAssets(assets || []);
      setCloudFindings(findings || []);
      setRawResults(raw || []);
    } catch (error) {
      console.error('Error fetching Katana Company results:', error);
      setError('Failed to load cloud assets, findings, and raw results');
    } finally {
      setIsLoading(false);
    }
  };

  const loadAllAvailableDomains = async () => {
    if (!activeTarget?.id) return;
    
    try {
      // Use the same endpoint as the config modal
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/consolidated-company-domains/${activeTarget.id}`
      );
      
      if (response.ok) {
        const data = await response.json();
        if (data.domains && Array.isArray(data.domains)) {
          setBaseDomains(data.domains);
        } else {
          setBaseDomains([]);
        }
      } else {
        console.warn('Failed to fetch consolidated company domains');
        setBaseDomains([]);
      }
    } catch (error) {
      console.error('Error fetching all available domains:', error);
      setBaseDomains([]);
    }
  };

  const fetchWildcardDomains = async () => {
    if (!activeTarget?.id) return;

    try {
      // Get all scope targets to find which root domains have been added as wildcard targets
      const scopeTargetsResponse = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/read`
      );
      
      if (!scopeTargetsResponse.ok) {
        throw new Error('Failed to fetch scope targets');
      }

      const scopeTargetsData = await scopeTargetsResponse.json();
      
      // Check if response is directly an array or has a targets property
      const targets = Array.isArray(scopeTargetsData) ? scopeTargetsData : scopeTargetsData.targets;
      
      // Ensure we have valid targets data
      if (!targets || !Array.isArray(targets)) {
        console.log('No valid targets data found:', scopeTargetsData);
        setWildcardDomains([]);
        return;
      }

      const wildcardTargets = targets.filter(target => {
        if (!target || target.type !== 'Wildcard') return false;
        
        // Remove *. prefix from wildcard target to match with base domains
        const rootDomainFromWildcard = target.scope_target.startsWith('*.') 
          ? target.scope_target.substring(2) 
          : target.scope_target;
        
        const isMatch = baseDomains.includes(rootDomainFromWildcard);
        
        return isMatch;
      });

      const wildcardDomainsData = [];

      // For each wildcard target, fetch its live web servers
      for (const wildcardTarget of wildcardTargets) {
        try {
          const liveWebServersResponse = await fetch(
            `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/scope-targets/${wildcardTarget.id}/target-urls`
          );

          if (liveWebServersResponse.ok) {
            const liveWebServersData = await liveWebServersResponse.json();
            
            // Check if response is directly an array or has a target_urls property
            const targetUrls = Array.isArray(liveWebServersData) ? liveWebServersData : liveWebServersData.target_urls;
            
            // Ensure we have valid target_urls data
            if (!targetUrls || !Array.isArray(targetUrls)) {
              continue;
            }

            const discoveredDomains = Array.from(new Set(
              targetUrls
                .map(url => {
                  try {
                    if (!url || !url.url) return null;
                    const urlObj = new URL(url.url);
                    return urlObj.hostname;
                  } catch {
                    return null;
                  }
                })
                .filter(domain => domain && domain !== wildcardTarget.scope_target)
            ));

            if (discoveredDomains.length > 0) {
              const rootDomainFromWildcard = wildcardTarget.scope_target.startsWith('*.') 
                ? wildcardTarget.scope_target.substring(2) 
                : wildcardTarget.scope_target;
              
              wildcardDomainsData.push({
                rootDomain: rootDomainFromWildcard,
                wildcardTarget: wildcardTarget.scope_target,
                discoveredDomains
              });
            }
          }
        } catch (error) {
          console.error(`Error fetching live web servers for ${wildcardTarget.scope_target}:`, error);
        }
      }

      setWildcardDomains(wildcardDomainsData);
    } catch (error) {
      console.error('Error fetching wildcard domains:', error);
      setWildcardDomains([]);
    }
  };

  const fetchLiveWebServers = async () => {
    if (!activeTarget?.id) return;

    try {
      // Fetch live web servers from IP port scans
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/ip-port-scans/${activeTarget.id}`
      );
      
      if (response.ok) {
        const data = await response.json();
        
        if (data && Array.isArray(data) && data.length > 0) {
          // Get the most recent scan
          const latestScan = data[0];
          
          if (latestScan && latestScan.scan_id) {
            // Fetch live web servers for the latest scan
            const liveWebServersResponse = await fetch(
              `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/live-web-servers/${latestScan.scan_id}`
            );
            
            if (liveWebServersResponse.ok) {
              const liveWebServersData = await liveWebServersResponse.json();
              
              if (liveWebServersData && Array.isArray(liveWebServersData)) {
                const urls = liveWebServersData.map(server => server.url).filter(url => url);
                setLiveWebServers(urls);
              } else {
                setLiveWebServers([]);
              }
            } else {
              setLiveWebServers([]);
            }
          } else {
            setLiveWebServers([]);
          }
        } else {
          setLiveWebServers([]);
        }
      } else {
        setLiveWebServers([]);
      }
    } catch (error) {
      console.error('Error fetching live web servers:', error);
      setLiveWebServers([]);
    }
  };

  const getServiceBadgeVariant = (service) => {
    if (service.includes('aws')) return 'warning';
    if (service.includes('gcp')) return 'info';
    if (service.includes('azure')) return 'primary';
    return 'secondary';
  };

  const getServiceName = (service) => {
    const parts = service.split('_');
    if (parts.length >= 2) {
      return `${parts[0].toUpperCase()} ${parts[1]}`;
    }
    return service.toUpperCase();
  };



  const loadDomainRawResults = async (domain) => {
    if (!activeTarget?.id) return;
    
    setLoadingDomains(prev => ({ ...prev, [domain]: true }));
    
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/katana-company/target/${activeTarget.id}/raw-results?domain=${encodeURIComponent(domain)}`
      );
      
      if (!response.ok) {
        throw new Error('Failed to fetch raw results for domain');
      }
      
      const results = await response.json();
      if (results.length > 0) {
        setLoadedRawResults(prev => ({ ...prev, [domain]: results[0] }));
      }
    } catch (error) {
      console.error('Error loading raw results for domain:', error);
      setError(`Failed to load raw results for ${domain}`);
    } finally {
      setLoadingDomains(prev => ({ ...prev, [domain]: false }));
    }
  };

  const clearDomainRawResults = (domain) => {
    setLoadedRawResults(prev => {
      const newResults = { ...prev };
      delete newResults[domain];
      return newResults;
    });
  };

  const clearAllRawResults = () => {
    setLoadedRawResults({});
  };

  const copyRawResults = async (domain) => {
    if (!loadedRawResults[domain]?.raw_output) return;
    
    setCopyingDomains(prev => ({ ...prev, [domain]: true }));
    
    try {
      await navigator.clipboard.writeText(loadedRawResults[domain].raw_output);
      // Show success feedback briefly
      setTimeout(() => {
        setCopyingDomains(prev => ({ ...prev, [domain]: false }));
      }, 1000);
    } catch (error) {
      console.error('Failed to copy raw results:', error);
      setCopyingDomains(prev => ({ ...prev, [domain]: false }));
      setError(`Failed to copy raw results for ${domain}`);
    }
  };

  const handleModalClose = () => {
    setActiveTab('assets');
    setError('');
    setRawResults([]);
    setAllAvailableDomains([]);
    setBaseDomains([]);
    setWildcardDomains([]);
    setLiveWebServers([]);
    setLoadedRawResults({});
    setLoadingDomains({});
    setCopyingDomains({});
    setLastLoadedScanId(null);
    handleClose();
  };

  return (
    <Modal 
      show={show} 
      onHide={handleModalClose} 
      size="xl" 
      backdrop={true}
      className="text-light"
      contentClassName="bg-dark border-secondary"
    >
      <Modal.Header closeButton className="bg-dark border-secondary">
        <Modal.Title className="text-light">
          Katana Company Scan Results - Cloud Asset Enumeration
          {activeTarget && (
            <div className="text-light fs-6 fw-normal mt-1" style={{ opacity: 0.8 }}>
              {activeTarget.scope_target}
            </div>
          )}
        </Modal.Title>
      </Modal.Header>
      <Modal.Body className="bg-dark text-light">
        {error && <Alert variant="danger" className="bg-danger bg-opacity-10 border-danger text-light">{error}</Alert>}
        
        {(mostRecentKatanaCompanyScan || rawResults.length > 0 || cloudAssets.length > 0 || cloudFindings.length > 0) && (
          <div className="mb-3">
            <Row>
              <Col md={6}>
                <Card className="h-100 bg-dark border-secondary">
                  <Card.Body>
                    <Card.Title className="fs-6 text-light">Scan Progress</Card.Title>
                    <div className="d-flex align-items-center">
                      <div className="flex-grow-1">
                        <p className="mb-1 text-light"><strong>Total Domains:</strong> {allAvailableDomains.length}</p>
                        <p className="mb-1 text-light"><strong>Scanned:</strong> <Badge bg="success">{rawResults.filter(r => r.has_been_scanned).length}</Badge></p>
                        <p className="mb-1 text-light"><strong>Not Scanned:</strong> <Badge bg="secondary">{allAvailableDomains.length - rawResults.filter(r => r.has_been_scanned).length}</Badge></p>
                        <p className="mb-0 text-light"><strong>Latest Scan:</strong> {mostRecentKatanaCompanyScan?.execution_time || 'N/A'}</p>
                      </div>
                      <div className="ms-3">
                        {allAvailableDomains.length > 0 && (
                          <div className="position-relative d-flex align-items-center justify-content-center" style={{ width: '80px', height: '80px' }}>
                            <svg width="80" height="80" viewBox="0 0 42 42" className="position-absolute">
                              <circle 
                                cx="21" 
                                cy="21" 
                                r="15.9" 
                                fill="transparent" 
                                stroke="#495057" 
                                strokeWidth="3"
                              />
                              <circle 
                                cx="21" 
                                cy="21" 
                                r="15.9" 
                                fill="transparent" 
                                stroke="#28a745" 
                                strokeWidth="3"
                                strokeDasharray={`${((rawResults.filter(r => r.has_been_scanned).length / allAvailableDomains.length) * 100).toFixed(1)} ${100 - ((rawResults.filter(r => r.has_been_scanned).length / allAvailableDomains.length) * 100).toFixed(1)}`}
                                strokeDashoffset="25"
                                transform="rotate(-90 21 21)"
                                style={{ transition: 'stroke-dasharray 0.6s ease-in-out' }}
                              />
                            </svg>
                            <div className="position-absolute text-center">
                              <div className="text-light fw-bold" style={{ fontSize: '14px' }}>
                                {Math.round((rawResults.filter(r => r.has_been_scanned).length / allAvailableDomains.length) * 100)}%
                              </div>
                              <div className="text-light" style={{ fontSize: '10px', opacity: 0.7 }}>
                                Complete
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  </Card.Body>
                </Card>
              </Col>
              <Col md={6}>
                <Card className="h-100 bg-dark border-secondary">
                  <Card.Body>
                    <Card.Title className="fs-6 text-light">Discovery Summary</Card.Title>
                    <div className="d-flex align-items-center">
                      <div className="flex-grow-1">
                    <p className="mb-1 text-light"><strong>Cloud Assets:</strong> {cloudAssets.length}</p>
                    <p className="mb-1 text-light"><strong>Cloud Findings:</strong> {cloudFindings.length}</p>
                        <p className="mb-1 text-light"><strong>AWS:</strong> <Badge bg="warning">{cloudAssets.filter(a => a.service.includes('aws')).length}</Badge></p>
                        <p className="mb-0 text-light"><strong>GCP:</strong> <Badge bg="info">{cloudAssets.filter(a => a.service.includes('gcp')).length}</Badge> <strong>Azure:</strong> <Badge bg="primary">{cloudAssets.filter(a => a.service.includes('azure')).length}</Badge></p>
                      </div>
                      <div className="ms-3">
                        {cloudAssets.length > 0 && (
                          <div className="position-relative d-flex align-items-center justify-content-center" style={{ width: '80px', height: '80px' }}>
                            <svg width="80" height="80" viewBox="0 0 42 42" className="position-absolute">
                              {(() => {
                                const awsCount = cloudAssets.filter(a => a.service.includes('aws')).length;
                                const gcpCount = cloudAssets.filter(a => a.service.includes('gcp')).length;
                                const azureCount = cloudAssets.filter(a => a.service.includes('azure')).length;
                                const total = cloudAssets.length;
                                
                                const awsPercent = (awsCount / total) * 100;
                                const gcpPercent = (gcpCount / total) * 100;
                                const azurePercent = (azureCount / total) * 100;
                                
                                let offset = 25; // Start at top
                                const elements = [];
                                
                                // AWS segment (orange)
                                if (awsCount > 0) {
                                  elements.push(
                                    <circle 
                                      key="aws"
                                      cx="21" 
                                      cy="21" 
                                      r="15.9" 
                                      fill="transparent" 
                                      stroke="#ffc107" 
                                      strokeWidth="3"
                                      strokeDasharray={`${awsPercent.toFixed(1)} ${100 - awsPercent.toFixed(1)}`}
                                      strokeDashoffset={offset}
                                      transform="rotate(-90 21 21)"
                                      style={{ transition: 'stroke-dasharray 0.6s ease-in-out' }}
                                    />
                                  );
                                  offset += awsPercent;
                                }
                                
                                // GCP segment (blue)
                                if (gcpCount > 0) {
                                  elements.push(
                                    <circle 
                                      key="gcp"
                                      cx="21" 
                                      cy="21" 
                                      r="15.9" 
                                      fill="transparent" 
                                      stroke="#0dcaf0" 
                                      strokeWidth="3"
                                      strokeDasharray={`${gcpPercent.toFixed(1)} ${100 - gcpPercent.toFixed(1)}`}
                                      strokeDashoffset={offset}
                                      transform="rotate(-90 21 21)"
                                      style={{ transition: 'stroke-dasharray 0.6s ease-in-out' }}
                                    />
                                  );
                                  offset += gcpPercent;
                                }
                                
                                // Azure segment (purple)
                                if (azureCount > 0) {
                                  elements.push(
                                    <circle 
                                      key="azure"
                                      cx="21" 
                                      cy="21" 
                                      r="15.9" 
                                      fill="transparent" 
                                      stroke="#6f42c1" 
                                      strokeWidth="3"
                                      strokeDasharray={`${azurePercent.toFixed(1)} ${100 - azurePercent.toFixed(1)}`}
                                      strokeDashoffset={offset}
                                      transform="rotate(-90 21 21)"
                                      style={{ transition: 'stroke-dasharray 0.6s ease-in-out' }}
                                    />
                                  );
                                }
                                
                                return elements;
                              })()}
                            </svg>
                            <div className="position-absolute text-center">
                              <div className="text-light fw-bold" style={{ fontSize: '16px' }}>
                                {cloudAssets.length}
                              </div>
                              <div className="text-light" style={{ fontSize: '10px', opacity: 0.7 }}>
                                Assets
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  </Card.Body>
                </Card>
              </Col>
            </Row>
          </div>
        )}



        <Tabs 
          activeKey={activeTab} 
          onSelect={(k) => setActiveTab(k)} 
          className="mb-3"
          variant="pills"
        >
          <Tab eventKey="assets" title={`Cloud Assets (${cloudAssets.length})`}>
            {isLoading ? (
              <div className="text-center py-4 text-light">Loading cloud assets...</div>
            ) : cloudAssets.length > 0 ? (
              <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
                <Table responsive size="sm" variant="dark" className="border-secondary">
                  <thead>
                    <tr>
                      <th className="text-light">Service</th>
                      <th className="text-light">Cloud Asset</th>
                      <th className="text-light">Source</th>
                    </tr>
                  </thead>
                  <tbody>
                    {cloudAssets.map((asset, index) => {
                      // The cloud asset FQDN is stored in asset.url, just remove the protocol
                      const cloudAssetFQDN = asset.url.replace(/^https?:\/\//, '');
                      
                      // Use the source_url field directly from the backend
                      const sourceURL = asset.source_url || asset.url;
                      
                      return (
                        <tr key={index}>
                          <td>
                            <Badge bg={getServiceBadgeVariant(asset.service)}>
                              {getServiceName(asset.service)}
                            </Badge>
                          </td>
                          <td>
                            <code className="text-warning">{cloudAssetFQDN}</code>
                          </td>
                          <td>
                            <a href={sourceURL} target="_blank" rel="noopener noreferrer" className="text-decoration-none text-info">
                              <code className="text-info small">{sourceURL}</code>
                            </a>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </Table>
              </div>
            ) : (
              <div className="text-center py-4 text-light" style={{ opacity: 0.7 }}>
                No cloud assets found.
              </div>
            )}
          </Tab>
          
          <Tab eventKey="findings" title={`Cloud Findings (${cloudFindings.length})`}>
            {isLoading ? (
              <div className="text-center py-4 text-light">Loading cloud findings...</div>
            ) : cloudFindings.length > 0 ? (
              <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
                <Accordion>
                    {cloudFindings.map((finding, index) => (
                    <Accordion.Item 
                      key={index} 
                      eventKey={index.toString()}
                      className="bg-dark border-secondary"
                      style={{ '--bs-accordion-bg': '#212529', '--bs-accordion-border-color': '#6c757d' }}
                    >
                      <Accordion.Header 
                        className="bg-dark"
                        style={{ '--bs-accordion-btn-bg': '#212529', '--bs-accordion-btn-color': '#fff', '--bs-accordion-active-bg': '#343a40', '--bs-accordion-active-color': '#fff' }}
                      >
                        <div className="w-100 pe-3">
                          <div className="d-flex justify-content-between align-items-start">
                            <div className="d-flex align-items-center gap-2">
                          <Badge bg={getServiceBadgeVariant(finding.cloud_service)}>
                            {finding.cloud_service.toUpperCase()}
                          </Badge>
                          <Badge bg="info">{finding.type}</Badge>
                            </div>
                            <div className="text-end">
                          <code className="text-warning">
                                {finding.content.length > 30 ? finding.content.substring(0, 30) + '...' : finding.content}
                          </code>
                            </div>
                          </div>
                          <div className="mt-1">
                            <small className="text-light" style={{ opacity: 0.7 }}>
                              Found on: <code className="text-info small">{finding.url}</code>
                            </small>
                          </div>
                        </div>
                      </Accordion.Header>
                      <Accordion.Body 
                        className="bg-dark text-light"
                        style={{ '--bs-accordion-body-bg': '#212529', '--bs-accordion-body-color': '#fff' }}
                      >
                        <div className="mb-3">
                          <strong>Source URL:</strong>
                          <br />
                          <a href={finding.url} target="_blank" rel="noopener noreferrer" className="text-decoration-none text-info">
                            <code className="text-info">{finding.url}</code>
                          </a>
                        </div>
                        
                        <div className="mb-3">
                          <strong>Finding:</strong>
                          <br />
                          <code className="text-warning">{finding.content}</code>
                        </div>
                        
                        {(finding.context_before || finding.context_after) && (
                          <div className="mb-3">
                            <strong>Context:</strong>
                            <div className="p-3 rounded mt-2" style={{ backgroundColor: '#f8f9fa', border: '1px solid #dee2e6' }}>
                              <code style={{ 
                                whiteSpace: 'pre-wrap', 
                                wordBreak: 'break-word',
                                color: '#495057',
                                backgroundColor: 'transparent',
                                fontSize: '0.875rem'
                              }}>
                                <span style={{ color: '#6c757d' }}>{finding.context_before}</span>
                                <mark className="bg-warning text-dark fw-bold">{finding.content}</mark>
                                <span style={{ color: '#6c757d' }}>{finding.context_after}</span>
                              </code>
                            </div>
                          </div>
                        )}
                        
                        <div className="text-light small" style={{ opacity: 0.7 }}>
                          <div><strong>Description:</strong> {finding.description}</div>
                          {finding.match_position && (
                            <div><strong>Position:</strong> Character {finding.match_position}</div>
                          )}
                          {finding.last_scanned_at && (
                            <div><strong>Last Scanned:</strong> {new Date(finding.last_scanned_at).toLocaleString()}</div>
                          )}
                        </div>
                      </Accordion.Body>
                    </Accordion.Item>
                  ))}
                </Accordion>
              </div>
            ) : (
              <div className="text-center py-4 text-light" style={{ opacity: 0.7 }}>
                No cloud findings found.
              </div>
            )}
          </Tab>

          <Tab eventKey="raw" title={`Raw Results (${rawResults.length})`}>
            {isLoading ? (
              <div className="text-center py-4 text-light">Loading raw results...</div>
            ) : rawResults.length > 0 ? (
              <div>
                <div className="mb-3 d-flex justify-content-end">
                  <Button
                    variant="outline-danger"
                    size="sm"
                    onClick={clearAllRawResults}
                    disabled={Object.keys(loadedRawResults).length === 0}
                  >
                    Clear All Raw Results
                  </Button>
                </div>
                <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
                  <Accordion>
                    {rawResults.map((result, index) => (
                      <Accordion.Item 
                        key={index} 
                        eventKey={index.toString()}
                        className="bg-dark border-secondary"
                        style={{ '--bs-accordion-bg': '#212529', '--bs-accordion-border-color': '#6c757d' }}
                      >
                        <Accordion.Header 
                          className="bg-dark"
                          style={{ '--bs-accordion-btn-bg': '#212529', '--bs-accordion-btn-color': '#fff', '--bs-accordion-active-bg': '#343a40', '--bs-accordion-active-color': '#fff' }}
                        >
                          <div className="w-100 pe-3">
                            <div className="d-flex justify-content-between align-items-center">
                              <div className="d-flex align-items-center gap-2">
                                <strong className="text-light">{result.domain}</strong>
                                <Badge bg={result.has_been_scanned ? "success" : "secondary"}>
                                  {result.has_been_scanned ? "Scanned" : "Not Scanned"}
                                </Badge>
                              </div>
                              <div className="text-end">
                                <small className="text-light" style={{ opacity: 0.7 }}>
                                  {result.has_been_scanned && result.last_scanned_at ? 
                                    `Last scanned: ${new Date(result.last_scanned_at).toLocaleString()}` : 
                                    "Never scanned"
                                  }
                                </small>
                              </div>
                            </div>
                          </div>
                        </Accordion.Header>
                        <Accordion.Body 
                          className="bg-dark text-light"
                          style={{ '--bs-accordion-body-bg': '#212529', '--bs-accordion-body-color': '#fff' }}
                        >
                          {result.has_been_scanned ? (
                            <>
                              <div className="mb-3">
                                <small className="text-light" style={{ opacity: 0.7 }}>
                                  <strong>Scan ID:</strong> {result.last_scan_id || 'N/A'}
                                </small>
                              </div>
                              
                              <div className="mb-3 d-flex gap-2">
                                {!loadedRawResults[result.domain] ? (
                                  <Button
                                    variant="outline-info"
                                    size="sm"
                                    onClick={() => loadDomainRawResults(result.domain)}
                                    disabled={loadingDomains[result.domain]}
                                  >
                                    {loadingDomains[result.domain] ? (
                                      <>
                                        <Spinner size="sm" className="me-1" />
                                        Loading...
                                      </>
                                    ) : (
                                      'Load Raw Results'
                                    )}
                                  </Button>
                                ) : (
                                  <>
                                    <Button
                                      variant="outline-success"
                                      size="sm"
                                      onClick={() => copyRawResults(result.domain)}
                                      disabled={copyingDomains[result.domain]}
                                    >
                                      {copyingDomains[result.domain] ? (
                                        <>
                                          <Spinner size="sm" className="me-1" />
                                          Copied!
                                        </>
                                      ) : (
                                        'Copy Raw Results'
                                      )}
                                    </Button>
                                    <Button
                                      variant="outline-warning"
                                      size="sm"
                                      onClick={() => clearDomainRawResults(result.domain)}
                                    >
                                      Clear Raw Results
                                    </Button>
                                  </>
                                )}
                              </div>

                              {loadedRawResults[result.domain] && (
                                <>
                                  <div className="mb-2">
                                    <small className="text-light" style={{ opacity: 0.7 }}>
                                      <strong>Output Size:</strong> {loadedRawResults[result.domain].raw_output ? loadedRawResults[result.domain].raw_output.length.toLocaleString() : 0} characters
                                    </small>
                                  </div>
                                  <div className="p-3" style={{ backgroundColor: '#1a1a1a', border: '1px solid #333', borderRadius: '4px' }}>
                                    <pre style={{ 
                                      whiteSpace: 'pre-wrap', 
                                      wordBreak: 'break-word',
                                      color: '#e9ecef',
                                      fontSize: '0.875rem',
                                      margin: 0,
                                      maxHeight: '400px',
                                      overflowY: 'auto'
                                    }}>
                                      {loadedRawResults[result.domain].raw_output || 'No raw output available'}
                                    </pre>
                                  </div>
                                </>
                              )}
                            </>
                          ) : (
                            <div className="text-center py-4">
                              <div className="text-light mb-2" style={{ opacity: 0.8 }}>
                                <i className="fa fa-clock-o me-2"></i>
                                This domain has not been scanned yet
                              </div>
                              <small className="text-light" style={{ opacity: 0.6 }}>
                                The domain was included in the scan configuration but may still be pending or may have encountered an error during scanning.
                              </small>
                            </div>
                          )}
                        </Accordion.Body>
                      </Accordion.Item>
                    ))}
                  </Accordion>
                </div>
              </div>
            ) : (
              <div className="text-center py-4 text-light" style={{ opacity: 0.7 }}>
                No raw scan results available.
              </div>
            )}
          </Tab>
        </Tabs>

        {!isLoading && cloudAssets.length === 0 && cloudFindings.length === 0 && mostRecentKatanaCompanyScan && (
          <Alert variant="info" className="bg-info bg-opacity-10 border-info text-light">
            <Alert.Heading className="text-light">No Cloud Assets Found</Alert.Heading>
            <p>The Katana scan completed but didn't discover any cloud assets or findings. This could mean:</p>
            <ul>
              <li>The scanned domains don't use cloud services</li>
              <li>Cloud assets are not publicly exposed</li>
              <li>The domains require authentication to access cloud resources</li>
              <li>Cloud assets are referenced in non-crawlable content</li>
            </ul>
          </Alert>
        )}
      </Modal.Body>
    </Modal>
  );
};

export default KatanaCompanyResultsModal; 