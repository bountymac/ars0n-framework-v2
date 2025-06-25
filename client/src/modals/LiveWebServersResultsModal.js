import { useState, useEffect } from 'react';
import { Modal, Button, Tab, Tabs, Table, Badge, Spinner, Alert } from 'react-bootstrap';
import { MdCopyAll } from 'react-icons/md';

const LiveWebServersResultsModal = ({ show, onHide, activeTarget, consolidatedNetworkRanges, mostRecentIPPortScan }) => {
  const [activeTab, setActiveTab] = useState('network-ranges');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [liveWebServers, setLiveWebServers] = useState([]);
  const [discoveredIPs, setDiscoveredIPs] = useState([]);
  const [scanData, setScanData] = useState(null);

  const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8443';

  useEffect(() => {
    if (show && mostRecentIPPortScan && mostRecentIPPortScan.scan_id) {
      fetchIPPortScanData();
    }
  }, [show, mostRecentIPPortScan]);

  const fetchIPPortScanData = async () => {
    if (!mostRecentIPPortScan || !mostRecentIPPortScan.scan_id) return;
    
    setLoading(true);
    console.log('LiveWebServersResultsModal: Starting fetchIPPortScanData for scanId:', mostRecentIPPortScan.scan_id);
    
    try {
      // Fetch scan info
      const scanResponse = await fetch(`${API_BASE_URL}/ip-port-scan/status/${mostRecentIPPortScan.scan_id}`);
      if (scanResponse.ok) {
        const scanInfo = await scanResponse.json();
        setScanData(scanInfo);
      }

      // Fetch live web servers
      const webServersResponse = await fetch(`${API_BASE_URL}/ip-port-scan/${mostRecentIPPortScan.scan_id}/live-web-servers`);
      if (webServersResponse.ok) {
        const webServers = await webServersResponse.json();
        setLiveWebServers(webServers || []);
      }

      // Fetch discovered IPs
      const ipsResponse = await fetch(`${API_BASE_URL}/ip-port-scan/${mostRecentIPPortScan.scan_id}/discovered-ips`);
      if (ipsResponse.ok) {
        const ips = await ipsResponse.json();
        setDiscoveredIPs(ips || []);
      }
    } catch (error) {
      console.error('LiveWebServersResultsModal: Error fetching IP/Port scan data:', error);
      setError('Failed to load IP/Port scan data');
    } finally {
      setLoading(false);
    }
  };

  const handleCopyText = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy text:', err);
    }
  };

  const getScanTypeBadgeVariant = (scanType) => {
    switch (scanType?.toLowerCase()) {
      case 'net':
        return 'primary';
      case 'netd':
        return 'info';
      case 'asn':
        return 'success';
      default:
        return 'secondary';
    }
  };

  const getSourceBadgeVariant = (source) => {
    switch (source?.toLowerCase()) {
      case 'amass_intel':
        return 'danger';
      case 'metabigor':
        return 'warning';
      case 'amass_intel, metabigor':
        return 'success';
      default:
        return 'secondary';
    }
  };

  const renderSourceBadges = (source) => {
    if (source === 'amass_intel, metabigor') {
      return (
        <div className="d-flex gap-1">
          <Badge bg="danger" className="small">Amass Intel</Badge>
          <Badge bg="warning" className="small">Metabigor</Badge>
        </div>
      );
    } else if (source === 'amass_intel') {
      return <Badge bg="danger">Amass Intel</Badge>;
    } else if (source === 'metabigor') {
      return <Badge bg="warning">Metabigor</Badge>;
    } else {
      return <Badge bg="secondary">{source}</Badge>;
    }
  };

  const getStatusColor = (statusCode) => {
    if (!statusCode) return 'secondary';
    if (statusCode >= 200 && statusCode < 300) return 'success';
    if (statusCode >= 300 && statusCode < 400) return 'info';
    if (statusCode >= 400 && statusCode < 500) return 'warning';
    if (statusCode >= 500) return 'danger';
    return 'secondary';
  };

  const formatResponseTime = (responseTime) => {
    if (!responseTime) return 'N/A';
    return `${responseTime}ms`;
  };

  const formatFileSize = (bytes) => {
    if (!bytes) return 'N/A';
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
  };

  return (
    <Modal show={show} onHide={onHide} size="xl" className="modal-90w">
      <Modal.Header closeButton className="bg-dark text-white">
        <Modal.Title>Live Web Servers Results</Modal.Title>
      </Modal.Header>
      <Modal.Body className="bg-dark text-white">
        <Tabs
          activeKey={activeTab}
          onSelect={(k) => setActiveTab(k)}
          className="mb-3"
          variant="pills"
        >
          <Tab eventKey="network-ranges" title={`Consolidated Network Ranges (${consolidatedNetworkRanges?.length || 0})`}>
            <div className="mb-3">
              <h5 className="text-danger">Consolidated Network Ranges</h5>
              <p className="text-white-50 small">
                Unique network ranges discovered from Amass Intel and Metabigor scans, deduplicated by CIDR block and ASN combination.
              </p>
            </div>

            {loading && (
              <div className="text-center py-4">
                <Spinner animation="border" variant="danger" />
                <p className="mt-2">Loading network ranges...</p>
              </div>
            )}

            {error && (
              <Alert variant="danger" className="mb-3">
                {error}
              </Alert>
            )}

            {!loading && !error && (
              <>
                <div className="mb-3">
                  <span className="text-white-50">
                    Total Network Ranges: <strong className="text-danger">{consolidatedNetworkRanges?.length || 0}</strong>
                  </span>
                </div>

                {consolidatedNetworkRanges && consolidatedNetworkRanges.length > 0 ? (
                  <div className="table-responsive">
                    <Table striped bordered hover variant="dark" className="mb-0">
                      <thead>
                        <tr>
                          <th>CIDR Block</th>
                          <th>ASN</th>
                          <th>Organization</th>
                          <th>Description/Scan Type</th>
                          <th>Country</th>
                          <th>Source</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {consolidatedNetworkRanges.map((range, index) => (
                          <tr key={index}>
                            <td>
                              <code className="text-danger">{range.cidr_block}</code>
                            </td>
                            <td>
                              <code className="text-info">{range.asn || 'N/A'}</code>
                            </td>
                            <td className="text-truncate" style={{ maxWidth: '200px' }}>
                              {range.organization || 'N/A'}
                            </td>
                            <td>
                              {range.source === 'amass_intel' ? (
                                <span className="text-white-50">{range.description || 'N/A'}</span>
                              ) : (
                                <Badge 
                                  bg={getScanTypeBadgeVariant(range.scan_type)}
                                  className="text-uppercase"
                                >
                                  {range.scan_type || 'N/A'}
                                </Badge>
                              )}
                            </td>
                            <td>
                              {range.country ? (
                                <Badge bg="secondary">{range.country}</Badge>
                              ) : (
                                <span className="text-white-50">N/A</span>
                              )}
                            </td>
                            <td>
                              {renderSourceBadges(range.source)}
                            </td>
                            <td>
                              <Button
                                variant="outline-danger"
                                size="sm"
                                onClick={() => handleCopyText(range.cidr_block)}
                                title="Copy CIDR block"
                              >
                                <MdCopyAll />
                              </Button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </Table>
                  </div>
                ) : (
                  <div className="text-center py-5">
                    <div className="text-white-50">
                      <h6>No consolidated network ranges found</h6>
                      <p className="small">
                        Run Amass Intel or Metabigor scans, then click "Consolidate" to see network ranges here.
                      </p>
                    </div>
                  </div>
                )}
              </>
            )}
          </Tab>

          <Tab eventKey="discovered-ips" title={`Discovered IPs (${discoveredIPs.length})`}>
            <div className="mb-3">
              <h5 className="text-danger">Discovered IPs</h5>
              <p className="text-white-50 small">
                Live IP addresses discovered within the consolidated network ranges via IP/Port scanning.
              </p>
            </div>

            {loading ? (
              <div className="text-center py-4">
                <Spinner animation="border" variant="danger" />
                <p className="mt-2">Loading discovered IPs...</p>
              </div>
            ) : (
              <div className="table-responsive">
                <Table striped bordered hover variant="dark" size="sm">
                  <thead>
                    <tr>
                      <th>IP Address</th>
                      <th>Hostname</th>
                      <th>Network Range</th>
                      <th>Discovered At</th>
                    </tr>
                  </thead>
                  <tbody>
                    {discoveredIPs.map((ip, index) => (
                      <tr key={index}>
                        <td><code className="text-info">{ip.ip_address}</code></td>
                        <td className="text-truncate" style={{ maxWidth: '250px' }} title={ip.hostname}>
                          {ip.hostname ? (
                            <code className="text-warning">{ip.hostname}</code>
                          ) : (
                            <span className="text-muted">N/A</span>
                          )}
                        </td>
                        <td><code className="text-warning">{ip.network_range}</code></td>
                        <td>{new Date(ip.discovered_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </Table>
                {discoveredIPs.length === 0 && (
                  <div className="text-center py-4 text-white-50">
                    {mostRecentIPPortScan ? 'No discovered IPs found.' : 'Run an IP/Port scan to see discovered IPs here.'}
                  </div>
                )}
              </div>
            )}
          </Tab>

          <Tab eventKey="live-servers" title={`Live Web Servers (${liveWebServers.length})`}>
            <div className="mb-3">
              <h5 className="text-danger">Live Web Servers</h5>
              <p className="text-white-50 small">
                Active web servers discovered within the consolidated network ranges via IP/Port scanning.
              </p>
            </div>

            {loading ? (
              <div className="text-center py-4">
                <Spinner animation="border" variant="danger" />
                <p className="mt-2">Loading live web servers...</p>
              </div>
            ) : (
              <div className="table-responsive">
                                  <Table striped bordered hover variant="dark" size="sm">
                    <thead>
                      <tr>
                        <th>URL</th>
                        <th>IP Address</th>
                        <th>Hostname</th>
                        <th>Port</th>
                        <th>Protocol</th>
                        <th>Status</th>
                        <th>Title</th>
                        <th>Server</th>
                        <th>Technologies</th>
                      </tr>
                    </thead>
                    <tbody>
                      {liveWebServers.map((server, index) => (
                        <tr key={index}>
                          <td>
                            <a 
                              href={server.url} 
                              target="_blank" 
                              rel="noopener noreferrer"
                              className="text-info text-decoration-none"
                              style={{ fontSize: '0.85em' }}
                            >
                              {server.url}
                            </a>
                          </td>
                          <td><code className="text-info">{server.ip_address}</code></td>
                          <td className="text-truncate" style={{ maxWidth: '200px' }} title={server.hostname}>
                            {server.hostname ? (
                              <code className="text-warning">{server.hostname}</code>
                            ) : (
                              <span className="text-muted">N/A</span>
                            )}
                          </td>
                          <td>{server.port}</td>
                          <td>{server.protocol}</td>
                          <td>
                            <Badge bg={getStatusColor(server.status_code)}>
                              {server.status_code || 'N/A'}
                            </Badge>
                          </td>
                          <td className="text-truncate" style={{ maxWidth: '200px' }} title={server.title}>
                            {server.title || 'N/A'}
                          </td>
                          <td className="text-truncate" style={{ maxWidth: '150px' }} title={server.server_header}>
                            {server.server_header || 'N/A'}
                          </td>
                          <td>
                            {server.technologies && server.technologies.length > 0 ? (
                              <div>
                                {server.technologies.slice(0, 2).map((tech, i) => (
                                  <Badge key={i} bg="secondary" className="me-1 mb-1">
                                    {tech}
                                  </Badge>
                                ))}
                                {server.technologies.length > 2 && (
                                  <Badge bg="outline-secondary">+{server.technologies.length - 2}</Badge>
                                )}
                              </div>
                            ) : (
                              'N/A'
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                {liveWebServers.length === 0 && (
                  <div className="text-center py-4 text-white-50">
                    {mostRecentIPPortScan ? 'No live web servers found.' : 'Run an IP/Port scan to see live web servers here.'}
                  </div>
                )}
              </div>
            )}
          </Tab>

          <Tab eventKey="metadata" title="Metadata">
            <div className="mb-3">
              <h5 className="text-danger">Metadata & Analysis</h5>
              <p className="text-white-50 small">
                Detailed metadata information for discovered web servers and assets.
              </p>
            </div>

            <div className="text-center py-5">
              <div className="text-white-50">
                <h6>Metadata Collection</h6>
                <p className="small">
                  This functionality will be implemented in a future update. It will show:
                </p>
                <ul className="list-unstyled small">
                  <li>• SSL/TLS certificate information</li>
                  <li>• HTTP response headers and security headers</li>
                  <li>• Technology stack detection details</li>
                  <li>• DNS records and subdomain information</li>
                  <li>• Content analysis and file discovery</li>
                </ul>
                <p className="small mt-3">
                  Use the "Gather Metadata" button to begin collecting detailed information.
                </p>
              </div>
            </div>
          </Tab>
        </Tabs>
      </Modal.Body>
      <Modal.Footer className="bg-dark">
        <Button variant="outline-danger" onClick={onHide}>
          Close
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default LiveWebServersResultsModal; 