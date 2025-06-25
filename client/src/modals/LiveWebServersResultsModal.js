import React, { useState } from 'react';
import { Modal, Button, Tab, Tabs, Table, Badge, Spinner, Alert } from 'react-bootstrap';
import { MdCopyAll } from 'react-icons/md';

const LiveWebServersResultsModal = ({ show, onHide, activeTarget, consolidatedNetworkRanges }) => {
  const [activeTab, setActiveTab] = useState('network-ranges');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

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
          <Tab eventKey="network-ranges" title="Consolidated Network Ranges">
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

          <Tab eventKey="live-servers" title="Live Web Servers">
            <div className="mb-3">
              <h5 className="text-danger">Live Web Servers</h5>
              <p className="text-white-50 small">
                Active web servers discovered within the consolidated network ranges.
              </p>
            </div>

            <div className="text-center py-5">
              <div className="text-white-50">
                <h6>Live Web Servers Discovery</h6>
                <p className="small">
                  This functionality will be implemented in a future update. It will show:
                </p>
                <ul className="list-unstyled small">
                  <li>• Live IP addresses within network ranges</li>
                  <li>• Open ports discovered via port scanning</li>
                  <li>• Active web servers (HTTP/HTTPS)</li>
                  <li>• Service information and response details</li>
                </ul>
                <p className="small mt-3">
                  Use the "IP/Port Scan" and "Gather Metadata" buttons to begin the discovery process.
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