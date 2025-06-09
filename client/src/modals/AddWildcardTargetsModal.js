import { useState } from 'react';
import { Modal, Table, Button, Badge, Spinner, Alert, OverlayTrigger, Tooltip } from 'react-bootstrap';

const AddWildcardTargetsModal = ({ 
  show, 
  handleClose, 
  consolidatedCompanyDomains = [], 
  onAddWildcardTarget,
  scopeTargets = [],
  fetchScopeTargets,
  investigateResults = []
}) => {
  const [addingDomains, setAddingDomains] = useState(new Set());
  const [addedDomains, setAddedDomains] = useState(new Set());
  const [error, setError] = useState('');

  const getInvestigateDataForDomain = (domain) => {
    return investigateResults.find(result => result.domain === domain);
  };

  const renderSSLStatus = (sslInfo) => {
    if (!sslInfo) {
      return <Badge bg="secondary" className="me-1">No SSL</Badge>;
    }

    const badges = [];
    
    if (sslInfo.is_expired) {
      badges.push(<Badge key="expired" bg="danger" className="me-1">Expired</Badge>);
    }
    
    if (sslInfo.is_self_signed) {
      badges.push(<Badge key="self-signed" bg="warning" className="me-1">Self-Signed</Badge>);
    }
    
    if (sslInfo.is_mismatched) {
      badges.push(<Badge key="mismatched" bg="warning" className="me-1">Mismatched</Badge>);
    }
    
    if (badges.length === 0) {
      const expiration = new Date(sslInfo.expiration);
      const daysUntilExpiry = Math.ceil((expiration - new Date()) / (1000 * 60 * 60 * 24));
      
      if (daysUntilExpiry < 30) {
        badges.push(<Badge key="expiring" bg="warning" className="me-1">Expires Soon</Badge>);
      } else {
        badges.push(<Badge key="valid" bg="success" className="me-1">Valid SSL</Badge>);
      }
    }
    
    return (
      <OverlayTrigger
        placement="top"
        overlay={
          <Tooltip>
            <div>Issuer: {sslInfo.issuer}</div>
            <div>Expires: {new Date(sslInfo.expiration).toLocaleDateString()}</div>
          </Tooltip>
        }
      >
        <span>{badges}</span>
      </OverlayTrigger>
    );
  };

  const renderHttpStatus = (httpInfo) => {
    if (!httpInfo) {
      return <span className="text-muted">No Response</span>;
    }

    let variant = 'secondary';
    if (httpInfo.status_code >= 200 && httpInfo.status_code < 300) {
      variant = 'success';
    } else if (httpInfo.status_code >= 300 && httpInfo.status_code < 400) {
      variant = 'info';
    } else if (httpInfo.status_code >= 400) {
      variant = 'danger';
    }

    return (
      <OverlayTrigger
        placement="top"
        overlay={
          <Tooltip>
            <div>Status: {httpInfo.status_code}</div>
            <div>Title: {httpInfo.title || 'N/A'}</div>
            <div>Server: {httpInfo.server || 'N/A'}</div>
          </Tooltip>
        }
      >
        <Badge bg={variant}>{httpInfo.status_code}</Badge>
      </OverlayTrigger>
    );
  };

  const renderCompanyMatch = (companyMatch, domain) => {
    if (companyMatch === undefined || companyMatch === null) {
      return <span className="text-muted">Unknown</span>;
    }
    
    return companyMatch ? (
      <Badge bg="success">
        <i className="bi bi-check-circle me-1"></i>
        Match
      </Badge>
    ) : (
      <Badge bg="secondary">
        <i className="bi bi-x-circle me-1"></i>
        No Match
      </Badge>
    );
  };

  const handleAddDomain = async (domain) => {
    if (addingDomains.has(domain) || addedDomains.has(domain)) return;
    
    setAddingDomains(prev => new Set([...prev, domain]));
    setError('');
    
    try {
      // Prepend *. to make it a wildcard format
      const wildcardDomain = `*.${domain}`;
      
      const response = await fetch(`${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/add`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          type: 'Wildcard',
          mode: 'Passive',
          scope_target: wildcardDomain,
          active: false,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to add wildcard target');
      }

      setAddedDomains(prev => new Set([...prev, domain]));
      
      // Refresh scope targets list to update the count
      try {
        await fetchScopeTargets();
      } catch (refreshError) {
        console.warn('Failed to refresh scope targets:', refreshError);
      }
      
    } catch (err) {
      setError(`Failed to add ${domain}: ${err.message}`);
    } finally {
      setAddingDomains(prev => {
        const newSet = new Set(prev);
        newSet.delete(domain);
        return newSet;
      });
    }
  };

  const isAlreadyTarget = (domain) => {
    return scopeTargets.some(target => 
      target && 
      target.type === 'Wildcard' && 
      target.scope_target && 
      (target.scope_target.toLowerCase() === domain.toLowerCase() ||
       target.scope_target.toLowerCase() === `*.${domain}`.toLowerCase())
    );
  };

  const getDomainStatus = (domain) => {
    if (!domain) return 'available';
    if (addedDomains.has(domain) || isAlreadyTarget(domain)) {
      return 'added';
    }
    if (addingDomains.has(domain)) {
      return 'adding';
    }
    return 'available';
  };

  const getSourceBadgeVariant = (source) => {
    const sourceVariants = {
      'google_dorking': 'primary',
      'reverse_whois': 'info',
      'ctl_company': 'success',
      'securitytrails_company': 'warning',
      'censys_company': 'danger',
      'github_recon': 'secondary',
      'shodan_company': 'dark',
      'consolidated': 'info'
    };
    return sourceVariants[source] || 'light';
  };

  const getSourceDisplayName = (source) => {
    const sourceNames = {
      'google_dorking': 'Google Dorking',
      'reverse_whois': 'Reverse Whois',
      'ctl_company': 'Certificate Transparency',
      'securitytrails_company': 'SecurityTrails',
      'censys_company': 'Censys',
      'github_recon': 'GitHub',
      'shodan_company': 'Shodan',
      'consolidated': 'Multiple Sources'
    };
    return sourceNames[source] || source;
  };

  const handleCloseModal = () => {
    setAddingDomains(new Set());
    setAddedDomains(new Set());
    setError('');
    handleClose();
  };

  return (
    <>
      <style>{`
        .modal-fullscreen .modal-dialog {
          max-width: 100vw !important;
          width: 100vw !important;
          height: 100vh !important;
          margin: 0 !important;
        }
        .modal-fullscreen .modal-content {
          height: 100vh !important;
          border-radius: 0 !important;
        }
        .modal-fullscreen .modal-body {
          overflow-y: auto !important;
          flex: 1 !important;
        }
      `}</style>
      <Modal 
        show={show} 
        onHide={handleCloseModal} 
        size="xl" 
        data-bs-theme="dark"
        dialogClassName="modal-fullscreen"
      >
        <Modal.Header closeButton>
          <Modal.Title className="text-danger">Add Wildcard Targets</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {error && (
            <Alert variant="danger" dismissible onClose={() => setError('')}>
              {error}
            </Alert>
          )}
          
          {consolidatedCompanyDomains.length === 0 ? (
            <div className="text-center py-4">
              <p className="text-white">No consolidated domains available.</p>
              <p className="text-white-50 small">
                Run the company domain discovery tools and consolidate the results first.
              </p>
            </div>
          ) : (
            <div>
              <div className="mb-3">
                <p className="text-white">
                  Found <strong>{consolidatedCompanyDomains.length}</strong> unique domains from company discovery tools. 
                  Select domains to add as Wildcard scope targets for subdomain enumeration.
                </p>
                {investigateResults.length === 0 && (
                  <Alert variant="info">
                    <i className="bi bi-info-circle me-2"></i>
                    <strong>Tip:</strong> Run the "Investigate" scan first to gather SSL, ASN, and HTTP information 
                    that will help you make better decisions about which domains to add as targets.
                  </Alert>
                )}
              </div>
              
              <Table striped bordered hover variant="dark" responsive>
                <thead>
                  <tr>
                    <th>Domain</th>
                    <th>IP Address</th>
                    <th>Title</th>
                    <th>SSL Status</th>
                    <th>ASN Provider</th>
                    <th>HTTP</th>
                    <th>Company Match</th>
                    <th>Status</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {consolidatedCompanyDomains.map((item, index) => {
                    const domain = typeof item === 'string' ? item : item.domain;
                    const source = typeof item === 'string' ? 'consolidated' : item.source;
                    const created_at = typeof item === 'string' ? new Date().toISOString() : item.created_at;
                    const investigateData = getInvestigateDataForDomain(domain);
                    
                    if (!domain) {
                      return <tr key={index}><td colSpan="9" className="text-warning">Invalid domain data: {JSON.stringify(item)}</td></tr>;
                    }
                    
                    const status = getDomainStatus(domain);
                    
                    return (
                      <tr key={index}>
                        <td className="text-white fw-bold">
                          <a 
                            href={`https://${domain}`} 
                            target="_blank" 
                            rel="noopener noreferrer"
                            className="text-white text-decoration-none"
                            style={{ cursor: 'pointer' }}
                          >
                            {domain}
                          </a>
                        </td>
                        <td className="text-white-50 font-monospace">{investigateData?.ip_address || 'N/A'}</td>
                        <td 
                          className="text-white-50" 
                          style={{ 
                            maxWidth: '200px', 
                            overflow: 'hidden', 
                            textOverflow: 'ellipsis', 
                            whiteSpace: 'nowrap' 
                          }}
                          title={investigateData?.http?.title || 'N/A'}
                        >
                          {investigateData?.http?.title || 'N/A'}
                        </td>
                        <td>{renderSSLStatus(investigateData?.ssl)}</td>
                        <td className="text-white-50">
                          {investigateData?.asn?.provider || 'Unknown'}
                        </td>
                        <td>{renderHttpStatus(investigateData?.http)}</td>
                        <td>{renderCompanyMatch(investigateData?.company_match, domain)}</td>
                        <td>
                          {status === 'added' && (
                            <Badge bg="success">
                              <i className="bi bi-check-circle me-1"></i>
                              Added
                            </Badge>
                          )}
                          {status === 'adding' && (
                            <Badge bg="warning">
                              <Spinner size="sm" className="me-1" />
                              Adding...
                            </Badge>
                          )}
                          {status === 'available' && !isAlreadyTarget(domain) && (
                            <Badge bg="secondary">Available</Badge>
                          )}
                          {status === 'available' && isAlreadyTarget(domain) && (
                            <Badge bg="info">Existing Target</Badge>
                          )}
                        </td>
                        <td className="text-center align-middle">
                          {status === 'available' && !isAlreadyTarget(domain) && (
                            <Button
                              variant="danger"
                              onClick={() => handleAddDomain(domain)}
                              disabled={addingDomains.has(domain)}
                              className="d-flex align-items-center justify-content-center"
                              style={{ width: '100px' }}
                            >
                              <i className="bi bi-plus-circle me-1"></i>
                              Add
                            </Button>
                          )}
                          {(status === 'added' || isAlreadyTarget(domain)) && (
                            <Button 
                              variant="secondary" 
                              disabled 
                              className="d-flex align-items-center justify-content-center"
                              style={{ width: '100px' }}
                            >
                              <i className="bi bi-check-circle me-1"></i>
                              Exists
                            </Button>
                          )}
                          {status === 'adding' && (
                            <Button 
                              variant="warning" 
                              disabled 
                              className="d-flex align-items-center justify-content-center"
                              style={{ width: '100px' }}
                            >
                              <Spinner size="sm" className="me-1" />
                            </Button>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </Table>
            </div>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={handleCloseModal}>
            Close
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default AddWildcardTargetsModal; 