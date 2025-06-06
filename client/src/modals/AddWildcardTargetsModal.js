import { useState } from 'react';
import { Modal, Table, Button, Badge, Spinner, Alert } from 'react-bootstrap';

const AddWildcardTargetsModal = ({ 
  show, 
  handleClose, 
  consolidatedCompanyDomains = [], 
  onAddWildcardTarget,
  scopeTargets = [],
  fetchScopeTargets
}) => {
  const [addingDomains, setAddingDomains] = useState(new Set());
  const [addedDomains, setAddedDomains] = useState(new Set());
  const [error, setError] = useState('');

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
    <Modal show={show} onHide={handleCloseModal} size="xl" data-bs-theme="dark">
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
            </div>
            
            <Table striped bordered hover variant="dark" responsive>
              <thead>
                <tr>
                  <th>Domain</th>
                  <th>Discovery Source</th>
                  <th>Discovered</th>
                  <th>Status</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {consolidatedCompanyDomains.map((item, index) => {
                  const domain = typeof item === 'string' ? item : item.domain;
                  const source = typeof item === 'string' ? 'consolidated' : item.source;
                  const created_at = typeof item === 'string' ? new Date().toISOString() : item.created_at;
                  
                  if (!domain) {
                    return <tr key={index}><td colSpan="5" className="text-warning">Invalid domain data: {JSON.stringify(item)}</td></tr>;
                  }
                  
                  const status = getDomainStatus(domain);
                  
                  return (
                    <tr key={index}>
                      <td className="text-white fw-bold">{domain}</td>
                      <td className="text-white-50">
                        {getSourceDisplayName(source)}
                      </td>
                      <td className="text-white-50 small">
                        {new Date(created_at).toLocaleDateString()}
                      </td>
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
                      <td>
                        {status === 'available' && !isAlreadyTarget(domain) && (
                          <Button
                            variant="danger"
                            size="sm"
                            onClick={() => handleAddDomain(domain)}
                            disabled={addingDomains.has(domain)}
                          >
                            <i className="bi bi-plus-circle me-1"></i>
                            Add Target
                          </Button>
                        )}
                        {(status === 'added' || isAlreadyTarget(domain)) && (
                          <Button variant="success" size="sm" disabled>
                            <i className="bi bi-check-circle me-1"></i>
                            Exists
                          </Button>
                        )}
                        {status === 'adding' && (
                          <Button variant="warning" size="sm" disabled>
                            <Spinner size="sm" className="me-1" />
                            Adding...
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
  );
};

export default AddWildcardTargetsModal; 