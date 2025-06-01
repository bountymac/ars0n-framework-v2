import React, { useState, useEffect } from 'react';
import { Modal, Table, Nav, Row, Col } from 'react-bootstrap';
import { MdCopyAll } from 'react-icons/md';

export const AmassIntelResultsModal = ({ 
  showAmassIntelResultsModal, 
  handleCloseAmassIntelResultsModal, 
  amassIntelResults,
  setShowToast 
}) => {
  const [activeTab, setActiveTab] = useState('domains');
  const [rootDomains, setRootDomains] = useState([]);
  const [whoisData, setWhoisData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (showAmassIntelResultsModal && amassIntelResults && amassIntelResults.scan_id) {
      fetchAmassIntelData(amassIntelResults.scan_id);
    } else if (showAmassIntelResultsModal && amassIntelResults) {
      // Handle case where scan failed or has no scan_id
      if (amassIntelResults.status === 'error') {
        setError('Scan failed. Please check the scan details and try again.');
        setLoading(false);
      } else if (!amassIntelResults.scan_id) {
        setError('No scan ID available for this scan.');
        setLoading(false);
      }
    }
  }, [showAmassIntelResultsModal, amassIntelResults]);

  const fetchAmassIntelData = async (scanId) => {
    setLoading(true);
    setError(null);
    setRootDomains([]);
    setWhoisData([]);

    try {
      // Fetch root domains
      const domainsResponse = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass-intel/${scanId}/domains`
      );
      
      if (domainsResponse.ok) {
        const domainsData = await domainsResponse.json();
        setRootDomains(Array.isArray(domainsData) ? domainsData : []);
      } else {
        console.warn('Failed to fetch root domains:', domainsResponse.status);
        setRootDomains([]);
      }

      // Fetch WHOIS data
      const whoisResponse = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/amass-intel/${scanId}/whois`
      );
      
      if (whoisResponse.ok) {
        const whoisDataResult = await whoisResponse.json();
        setWhoisData(Array.isArray(whoisDataResult) ? whoisDataResult : []);
      } else {
        console.warn('Failed to fetch WHOIS data:', whoisResponse.status);
        setWhoisData([]);
      }

    } catch (error) {
      console.error('Error fetching Amass Intel data:', error);
      setError('Failed to fetch scan results. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleCopyDomain = async (domain) => {
    try {
      await navigator.clipboard.writeText(domain);
      setShowToast(true);
      setTimeout(() => setShowToast(false), 3000);
    } catch (err) {
      console.error('Failed to copy domain:', err);
    }
  };

  const getResultCount = () => {
    if (!amassIntelResults || amassIntelResults.status === 'error') return 0;
    
    try {
      if (amassIntelResults.result && typeof amassIntelResults.result === 'string') {
        const lines = amassIntelResults.result.split('\n').filter(line => line.trim());
        return lines.length;
      }
      return rootDomains.length;
    } catch (error) {
      console.error('Error calculating result count:', error);
      return 0;
    }
  };

  const getExecutionTime = () => {
    if (!amassIntelResults || !amassIntelResults.execution_time) return 'N/A';
    return amassIntelResults.execution_time;
  };

  const getScanStatus = () => {
    if (!amassIntelResults) return 'Unknown';
    return amassIntelResults.status || 'Unknown';
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    try {
      return new Date(dateString).toLocaleString();
    } catch (error) {
      return 'Invalid Date';
    }
  };

  const renderContent = () => {
    // Handle scan failure
    if (amassIntelResults && amassIntelResults.status === 'error') {
      return (
        <div className="text-center py-5">
          <div className="text-danger mb-3">
            <i className="bi bi-exclamation-triangle" style={{ fontSize: '3rem' }}></i>
          </div>
          <h5 className="text-danger mb-3">Scan Failed</h5>
          <p className="text-muted mb-3">
            The Amass Intel scan encountered an error and could not complete successfully.
          </p>
          {amassIntelResults.stderr && (
            <div className="alert alert-danger small text-start mt-3">
              <strong>Error Details:</strong><br />
              {amassIntelResults.stderr}
            </div>
          )}
        </div>
      );
    }

    // Handle loading state
    if (loading) {
      return (
        <div className="text-center py-5">
          <div className="spinner-border text-danger mb-3" role="status">
            <span className="visually-hidden">Loading...</span>
          </div>
          <p className="text-muted">Loading scan results...</p>
        </div>
      );
    }

    // Handle error state
    if (error) {
      return (
        <div className="text-center py-5">
          <div className="text-warning mb-3">
            <i className="bi bi-exclamation-triangle" style={{ fontSize: '3rem' }}></i>
          </div>
          <h5 className="text-warning mb-3">Error</h5>
          <p className="text-muted">{error}</p>
        </div>
      );
    }

    // Handle no results
    if (rootDomains.length === 0 && whoisData.length === 0) {
      return (
        <div className="text-center py-5">
          <div className="text-muted mb-3">
            <i className="bi bi-search" style={{ fontSize: '3rem' }}></i>
          </div>
          <h5 className="text-muted mb-3">No Results Found</h5>
          <p className="text-muted">
            No root domains were discovered for this company. This could be due to:
          </p>
          <ul className="list-unstyled text-muted small">
            <li>• Company name not found in WHOIS records</li>
            <li>• No certificates registered to this organization</li>
            <li>• Network timeout during scan</li>
          </ul>
        </div>
      );
    }

    // Render normal results
    return (
      <>
        <Nav variant="tabs" className="mb-3">
          <Nav.Item>
            <Nav.Link 
              active={activeTab === 'domains'} 
              onClick={() => setActiveTab('domains')}
              className={activeTab === 'domains' ? 'text-danger' : 'text-white'}
            >
              Root Domains ({rootDomains.length})
            </Nav.Link>
          </Nav.Item>
          <Nav.Item>
            <Nav.Link 
              active={activeTab === 'whois'} 
              onClick={() => setActiveTab('whois')}
              className={activeTab === 'whois' ? 'text-danger' : 'text-white'}
            >
              WHOIS Data ({whoisData.length})
            </Nav.Link>
          </Nav.Item>
        </Nav>

        {activeTab === 'domains' && (
          <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
            {rootDomains.length > 0 ? (
              <Table striped bordered hover variant="dark" className="mb-0">
                <thead>
                  <tr>
                    <th>Domain</th>
                    <th>Source</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {rootDomains.map((domain, index) => (
                    <tr key={index}>
                      <td className="font-monospace">{domain.domain}</td>
                      <td>
                        <span className="badge bg-secondary">{domain.source || 'intel'}</span>
                      </td>
                      <td>
                        <button 
                          onClick={() => handleCopyDomain(domain.domain)}
                          className="btn btn-sm btn-outline-danger"
                          title="Copy domain"
                        >
                          <MdCopyAll size={14} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            ) : (
              <div className="text-center py-4">
                <p className="text-muted">No root domains found.</p>
              </div>
            )}
          </div>
        )}

        {activeTab === 'whois' && (
          <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
            {whoisData.length > 0 ? (
              <Table striped bordered hover variant="dark" className="mb-0">
                <thead>
                  <tr>
                    <th>Domain</th>
                    <th>Registrant</th>
                    <th>Organization</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {whoisData.map((whois, index) => (
                    <tr key={index}>
                      <td className="font-monospace">{whois.domain}</td>
                      <td>{whois.registrant || 'N/A'}</td>
                      <td>{whois.organization || 'N/A'}</td>
                      <td>
                        <button 
                          onClick={() => handleCopyDomain(whois.domain)}
                          className="btn btn-sm btn-outline-danger"
                          title="Copy domain"
                        >
                          <MdCopyAll size={14} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            ) : (
              <div className="text-center py-4">
                <p className="text-muted">No WHOIS data found.</p>
              </div>
            )}
          </div>
        )}
      </>
    );
  };

  return (
    <Modal 
      data-bs-theme="dark" 
      show={showAmassIntelResultsModal} 
      onHide={handleCloseAmassIntelResultsModal} 
      size="xl"
      dialogClassName="modal-90w"
    >
      <Modal.Header closeButton>
        <Modal.Title className="text-danger">
          Amass Intel Results
          {amassIntelResults && (
            <span className="text-white fs-6 ms-3">
              Company: {amassIntelResults.company_name}
            </span>
          )}
        </Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {amassIntelResults && (
          <Row className="mb-3">
            <Col md={3}>
              <small className="text-white-50">Status:</small>
              <div className={`text-${getScanStatus() === 'success' ? 'success' : getScanStatus() === 'error' ? 'danger' : 'warning'}`}>
                {getScanStatus()}
              </div>
            </Col>
            <Col md={3}>
              <small className="text-white-50">Execution Time:</small>
              <div className="text-white">{getExecutionTime()}</div>
            </Col>
            <Col md={3}>
              <small className="text-white-50">Root Domains:</small>
              <div className="text-danger fw-bold">{rootDomains.length}</div>
            </Col>
            <Col md={3}>
              <small className="text-white-50">WHOIS Records:</small>
              <div className="text-info fw-bold">{whoisData.length}</div>
            </Col>
          </Row>
        )}

        {renderContent()}
      </Modal.Body>
    </Modal>
  );
};

export const AmassIntelHistoryModal = ({ 
  showAmassIntelHistoryModal, 
  handleCloseAmassIntelHistoryModal, 
  amassIntelScans 
}) => {
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const getStatusBadge = (status) => {
    const statusColors = {
      'completed': 'success',
      'running': 'primary',
      'pending': 'warning',
      'failed': 'danger'
    };
    return <span className={`badge bg-${statusColors[status] || 'secondary'}`}>{status}</span>;
  };

  return (
    <Modal 
      show={showAmassIntelHistoryModal} 
      onHide={handleCloseAmassIntelHistoryModal} 
      size="xl"
      className="text-white"
    >
      <Modal.Header closeButton className="bg-dark border-danger">
        <Modal.Title className="text-danger">Amass Intel Scan History</Modal.Title>
      </Modal.Header>
      <Modal.Body className="bg-dark">
        {amassIntelScans && amassIntelScans.length > 0 ? (
          <Table striped bordered hover variant="dark" className="mb-0">
            <thead>
              <tr>
                <th>Company Name</th>
                <th>Status</th>
                <th>Execution Time</th>
                <th>Root Domains Found</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody>
              {amassIntelScans.map((scan, index) => (
                <tr key={index}>
                  <td>{scan.company_name}</td>
                  <td>{getStatusBadge(scan.status)}</td>
                  <td>{scan.execution_time || 'N/A'}</td>
                  <td>
                    {scan.result && scan.status === 'completed' 
                      ? (() => {
                          try {
                            return JSON.parse(scan.result).length;
                          } catch {
                            return 'Error parsing results';
                          }
                        })()
                      : 'N/A'
                    }
                  </td>
                  <td>{formatDate(scan.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        ) : (
          <div className="text-center py-4">
            <p className="text-muted">No Amass Intel scans found.</p>
          </div>
        )}
      </Modal.Body>
    </Modal>
  );
}; 