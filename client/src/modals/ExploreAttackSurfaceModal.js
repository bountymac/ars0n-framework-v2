import React, { useState, useEffect } from 'react';
import { Modal, Button, Form, Table, Badge, Row, Col, Alert, Nav, InputGroup } from 'react-bootstrap';
import fetchAttackSurfaceAssets from '../utils/fetchAttackSurfaceAssets';

const ExploreAttackSurfaceModal = ({ 
  show, 
  handleClose, 
  activeTarget 
}) => {
  const [attackSurfaceAssets, setAttackSurfaceAssets] = useState([]);
  const [filteredAssets, setFilteredAssets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [sortColumn, setSortColumn] = useState('asset_type');
  const [sortDirection, setSortDirection] = useState('asc');
  const [activeTab, setActiveTab] = useState('asn');
  const [filters, setFilters] = useState([{ searchTerm: '', isNegative: false }]);

  const assetTypes = [
    { key: 'asn', label: 'Autonomous System Numbers (ASNs)', count: 0 },
    { key: 'network_range', label: 'Network Ranges', count: 0 },
    { key: 'ip_address', label: 'IP Addresses', count: 0 },
    { key: 'fqdn', label: 'Domain Names', count: 0 },
    { key: 'cloud_asset', label: 'Cloud Asset Domains', count: 0 },
    { key: 'live_web_server', label: 'Live Web Servers', count: 0 }
  ];

  useEffect(() => {
    if (show && activeTarget) {
      loadAttackSurfaceAssets();
    }
  }, [show, activeTarget]);

  useEffect(() => {
    applyFiltersAndSort();
  }, [attackSurfaceAssets, filters, sortColumn, sortDirection, activeTab]);

  const loadAttackSurfaceAssets = async () => {
    if (!activeTarget) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const data = await fetchAttackSurfaceAssets(activeTarget);
      setAttackSurfaceAssets(data.assets || []);
    } catch (err) {
      setError('Failed to load attack surface assets');
      console.error('Error loading attack surface assets:', err);
    } finally {
      setLoading(false);
    }
  };

  const getAssetTypeCounts = () => {
    const counts = {};
    assetTypes.forEach(type => {
      counts[type.key] = attackSurfaceAssets.filter(asset => asset.asset_type === type.key).length;
    });
    return counts;
  };

  const applyFiltersAndSort = () => {
    let filtered = [...attackSurfaceAssets];

    filtered = filtered.filter(asset => asset.asset_type === activeTab);

    const activeFilters = filters.filter(filter => filter.searchTerm.trim() !== '');
    
    if (activeFilters.length > 0) {
      filtered = filtered.filter(asset => {
        const searchableFields = [
          asset.asset_identifier,
          asset.asn_number,
          asset.asn_organization,
          asset.asn_description,
          asset.asn_country,
          asset.cidr_block,
          asset.ip_address,
          asset.ip_type,
          asset.url,
          asset.domain,
          asset.port?.toString(),
          asset.status_code?.toString(),
          asset.title,
          asset.web_server,
          asset.cloud_provider,
          asset.cloud_service_type,
          asset.cloud_region,
          asset.fqdn,
          asset.root_domain,
          asset.subdomain,
          asset.registrar,
          asset.creation_date,
          asset.expiration_date,
          asset.ssl_expiry_date,
          asset.ssl_issuer,
          asset.ssl_subject,
          asset.ssl_version,
          asset.ssl_cipher_suite,
          asset.resolved_ips?.join(' '),
          asset.name_servers?.join(' '),
          asset.mail_servers?.join(' '),
          asset.spf_record,
          asset.dkim_record,
          asset.dmarc_record,
          asset.caa_records?.join(' '),
          asset.txt_records?.join(' '),
          asset.mx_records?.join(' '),
          asset.ns_records?.join(' '),
          asset.a_records?.join(' '),
          asset.aaaa_records?.join(' '),
          asset.cname_records?.join(' '),
          asset.ptr_records?.join(' '),
          asset.srv_records?.join(' '),
          asset.ssl_protocols?.join(' '),
          asset.status?.join(' '),
          asset.technologies?.join(' ')
        ].filter(Boolean).join(' ').toLowerCase();
        
        return activeFilters.every(filter => {
          const assetContainsSearch = searchableFields.includes(filter.searchTerm.toLowerCase());
          return filter.isNegative ? !assetContainsSearch : assetContainsSearch;
        });
      });
    }

    filtered.sort((a, b) => {
      let aValue = a[sortColumn];
      let bValue = b[sortColumn];

      if (aValue === null || aValue === undefined) aValue = '';
      if (bValue === null || bValue === undefined) bValue = '';

      if (typeof aValue === 'string') aValue = aValue.toLowerCase();
      if (typeof bValue === 'string') bValue = bValue.toLowerCase();

      if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    setFilteredAssets(filtered);
  };

  const addSearchFilter = () => {
    setFilters([...filters, { searchTerm: '', isNegative: false }]);
  };

  const removeSearchFilter = (index) => {
    if (filters.length > 1) {
      const newFilters = filters.filter((_, i) => i !== index);
      setFilters(newFilters);
    }
  };

  const updateSearchFilter = (index, field, value) => {
    const newFilters = [...filters];
    newFilters[index][field] = value;
    setFilters(newFilters);
  };

  const handleFilterChange = (filterKey, value) => {
    setFilters(prev => ({
      ...prev,
      [filterKey]: value
    }));
  };

  const clearFilters = () => {
    setFilters([{ searchTerm: '', isNegative: false }]);
  };

  const handleSort = (column) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const renderSortIcon = (column) => {
    if (sortColumn !== column) return null;
    return sortDirection === 'asc' ? ' ↑' : ' ↓';
  };

  const getAssetTypeBadgeVariant = (assetType) => {
    switch (assetType) {
      case 'asn': return 'primary';
      case 'network_range': return 'secondary';
      case 'ip_address': return 'info';
      case 'live_web_server': return 'success';
      case 'cloud_asset': return 'warning';
      case 'fqdn': return 'danger';
      default: return 'dark';
    }
  };

  const getAssetTypeDisplayName = (assetType) => {
    switch (assetType) {
      case 'asn': return 'ASNs';
      case 'network_range': return 'Network Ranges';
      case 'ip_address': return 'IP Addresses';
      case 'live_web_server': return 'Live Web Servers';
      case 'cloud_asset': return 'Cloud Asset Domains';
      case 'fqdn': return 'Domain Names';
      default: return assetType;
    }
  };

  const renderAssetDetails = (asset) => {
    switch (asset.asset_type) {
      case 'asn':
        return (
          <div>
            <div className="mb-1"><span className="text-danger fw-bold">ASN:</span> {asset.asn_number}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Organization:</span> {asset.asn_organization}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Country:</span> {asset.asn_country}</div>
            {asset.asn_description && <div className="mb-1"><span className="text-danger fw-bold">Description:</span> {asset.asn_description}</div>}
          </div>
        );
      
      case 'network_range':
        return (
          <div>
            <div className="mb-1"><span className="text-danger fw-bold">CIDR:</span> {asset.cidr_block}</div>
          </div>
        );
      
      case 'ip_address':
        return (
          <div>
            <div className="mb-1"><span className="text-danger fw-bold">IP:</span> {asset.ip_address}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Type:</span> {asset.ip_type}</div>
          </div>
        );
      
      case 'live_web_server':
        return (
          <div>
            <div className="mb-1"><span className="text-danger fw-bold">URL:</span> {asset.url}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Domain:</span> {asset.domain}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Port:</span> {asset.port}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Protocol:</span> {asset.protocol}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Status:</span> {asset.status_code}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Title:</span> {asset.title}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Web Server:</span> {asset.web_server}</div>
            {asset.technologies && asset.technologies.length > 0 && (
              <div className="mb-1"><span className="text-danger fw-bold">Technologies:</span> {asset.technologies.join(', ')}</div>
            )}
          </div>
        );
      
      case 'cloud_asset':
        return (
          <div>
            <div className="mb-1"><span className="text-danger fw-bold">Provider:</span> {asset.cloud_provider}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Service:</span> {asset.cloud_service_type}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Region:</span> {asset.cloud_region}</div>
            <div className="mb-1"><span className="text-danger fw-bold">Identifier:</span> {asset.asset_identifier}</div>
          </div>
        );
      
      case 'fqdn':
        return (
          <div>
            <div className="mb-1"><span className="text-danger fw-bold">FQDN:</span> {asset.fqdn}</div>
            {asset.root_domain && <div className="mb-1"><span className="text-danger fw-bold">Root Domain:</span> {asset.root_domain}</div>}
            {asset.subdomain && <div className="mb-1"><span className="text-danger fw-bold">Subdomain:</span> {asset.subdomain}</div>}
            {asset.registrar && <div className="mb-1"><span className="text-danger fw-bold">Registrar:</span> {asset.registrar}</div>}
            {asset.creation_date && <div className="mb-1"><span className="text-danger fw-bold">Creation Date:</span> {asset.creation_date}</div>}
            {asset.expiration_date && <div className="mb-1"><span className="text-danger fw-bold">Expiration Date:</span> {asset.expiration_date}</div>}
            {asset.ssl_expiry_date && <div className="mb-1"><span className="text-danger fw-bold">SSL Expiry:</span> {asset.ssl_expiry_date}</div>}
            {asset.ssl_issuer && <div className="mb-1"><span className="text-danger fw-bold">SSL Issuer:</span> {asset.ssl_issuer}</div>}
            {asset.resolved_ips && asset.resolved_ips.length > 0 && (
              <div className="mb-1"><span className="text-danger fw-bold">Resolved IPs:</span> {asset.resolved_ips.join(', ')}</div>
            )}
            {asset.name_servers && asset.name_servers.length > 0 && (
              <div className="mb-1"><span className="text-danger fw-bold">Name Servers:</span> {asset.name_servers.join(', ')}</div>
            )}
            {asset.mail_servers && asset.mail_servers.length > 0 && (
              <div className="mb-1"><span className="text-danger fw-bold">Mail Servers:</span> {asset.mail_servers.join(', ')}</div>
            )}
            {asset.spf_record && <div className="mb-1"><span className="text-danger fw-bold">SPF:</span> {asset.spf_record}</div>}
            {asset.dkim_record && <div className="mb-1"><span className="text-danger fw-bold">DKIM:</span> {asset.dkim_record}</div>}
            {asset.dmarc_record && <div className="mb-1"><span className="text-danger fw-bold">DMARC:</span> {asset.dmarc_record}</div>}
          </div>
        );
      
      default:
        return <div className="mb-1"><span className="text-danger fw-bold">Identifier:</span> {asset.asset_identifier}</div>;
    }
  };

  const renderRelationships = (asset) => {
    if (!asset.relationships || asset.relationships.length === 0) {
      return <span className="text-white-50">None</span>;
    }

    return (
      <div>
        {asset.relationships.map((rel, index) => (
          <Badge key={index} variant="outline-info" className="me-1">
            {rel.relationship_type}
          </Badge>
        ))}
      </div>
    );
  };

  const renderFiltersForTab = () => {
    return (
      <Row className="g-3">
        <Col>
          <div className="mb-3">
            <div className="d-flex justify-content-between align-items-center mb-2">
              <Form.Label className="text-white small mb-0">Search Filters</Form.Label>
              <div>
                <Button 
                  variant="outline-success" 
                  size="sm" 
                  onClick={addSearchFilter}
                  className="me-2"
                >
                  Add Filter
                </Button>
                <Button 
                  variant="outline-danger" 
                  size="sm" 
                  onClick={clearFilters}
                >
                  Clear Filters
                </Button>
              </div>
            </div>
            {filters.map((filter, index) => (
              <div key={index} className={index > 0 ? "mt-2" : ""}>
                <InputGroup>
                  <Form.Control
                    type="text"
                    placeholder="Search across all data points (ASN, IP, domain, organization, etc.)..."
                    value={filter.searchTerm}
                    onChange={(e) => updateSearchFilter(index, 'searchTerm', e.target.value)}
                    data-bs-theme="dark"
                  />
                  <InputGroup.Text className="bg-dark border-secondary">
                    <Form.Check
                      type="checkbox"
                      id={`negative-search-checkbox-${index}`}
                      label="Negative Search"
                      checked={filter.isNegative}
                      onChange={(e) => updateSearchFilter(index, 'isNegative', e.target.checked)}
                      className="text-white-50 small m-0"
                      disabled={!filter.searchTerm}
                    />
                  </InputGroup.Text>
                  {filter.searchTerm && (
                    <Button 
                      variant="outline-secondary" 
                      onClick={() => updateSearchFilter(index, 'searchTerm', '')}
                      title="Clear this search"
                    >
                      ×
                    </Button>
                  )}
                  {filters.length > 1 && (
                    <Button 
                      variant="outline-danger" 
                      onClick={() => removeSearchFilter(index)}
                      title="Remove this filter"
                    >
                      🗑️
                    </Button>
                  )}
                </InputGroup>
              </div>
            ))}
          </div>
        </Col>
      </Row>
    );
  };

  const renderTableHeaders = () => {
    return [
      {
        key: 'asset_identifier',
        label: 'Identifier',
        sortable: true
      },
      {
        key: 'details',
        label: 'Details',
        sortable: false
      },
      {
        key: 'relationships',
        label: 'Relationships',
        sortable: false
      },
      {
        key: 'last_updated',
        label: 'Last Updated',
        sortable: true
      }
    ];
  };

  const counts = getAssetTypeCounts();

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
        .nav-tabs .nav-link {
          color: #6c757d;
          border: none;
          border-bottom: 2px solid transparent;
        }
        .nav-tabs .nav-link:hover {
          color: #fff;
          border-bottom-color: #6c757d;
        }
        .nav-tabs .nav-link.active {
          color: #dc3545;
          border-bottom-color: #dc3545;
          background: transparent;
        }
        .nav-tabs {
          display: flex;
          width: 100%;
        }
        .nav-tabs .nav-item {
          flex: 1;
          text-align: center;
        }
        .nav-tabs .nav-link {
          width: 100%;
          text-align: center;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
      `}</style>
      <Modal 
        show={show} 
        onHide={handleClose} 
        size="xl" 
        data-bs-theme="dark"
        dialogClassName="modal-fullscreen"
      >
        <Modal.Header closeButton>
          <Modal.Title className="text-danger">Explore Attack Surface - {activeTarget?.scope_target}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {loading && (
            <div className="text-center py-4">
              <div className="spinner-border text-danger" role="status">
                <span className="visually-hidden">Loading...</span>
              </div>
              <p className="mt-2 text-white">Loading attack surface assets...</p>
            </div>
          )}

          {error && (
            <Alert variant="danger" dismissible onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          {!loading && !error && (
            <>
              <Nav variant="tabs" className="mb-4" activeKey={activeTab} onSelect={(k) => setActiveTab(k)}>
                {assetTypes.map((type) => (
                  <Nav.Item key={type.key}>
                    <Nav.Link eventKey={type.key}>
                      {type.label} ({counts[type.key] || 0})
                    </Nav.Link>
                  </Nav.Item>
                ))}
              </Nav>

              <div className="mb-4 p-3 bg-dark rounded border">
                <div className="d-flex justify-content-between align-items-center mb-3">
                  <h6 className="text-white mb-0">
                    <i className="bi bi-funnel me-2"></i>
                    Filter Results
                  </h6>
                  <small className="text-white-50">
                    Showing {filteredAssets.length} of {counts[activeTab] || 0} assets
                    {(() => {
                      const activeFilters = filters.filter(filter => filter.searchTerm.trim() !== '');
                      if (activeFilters.length > 0) {
                        const filterDescriptions = activeFilters.map(filter => 
                          `${filter.isNegative ? 'excluding' : 'including'} "${filter.searchTerm}"`
                        );
                        return (
                          <span className="text-warning">
                            {' '}({filterDescriptions.join(', ')})
                          </span>
                        );
                      }
                      return null;
                    })()}
                  </small>
                </div>
                {renderFiltersForTab()}
              </div>

              <div className="table-responsive" style={{ maxHeight: '60vh', overflowY: 'auto' }}>
                <Table striped bordered hover variant="dark" responsive>
                  <thead>
                    <tr>
                      {renderTableHeaders().map((header) => (
                        <th 
                          key={header.key}
                          style={{ cursor: header.sortable ? 'pointer' : 'default', userSelect: 'none' }}
                          onClick={header.sortable ? () => handleSort(header.key) : undefined}
                        >
                          {header.label} {header.sortable && renderSortIcon(header.key)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {filteredAssets.map((asset) => (
                      <tr key={asset.id}>
                        <td>
                          <code>{asset.asset_identifier}</code>
                        </td>
                        <td>
                          <div style={{ maxWidth: '300px', fontSize: '0.875rem' }}>
                            {renderAssetDetails(asset)}
                          </div>
                        </td>
                        <td>
                          {renderRelationships(asset)}
                        </td>
                        <td>
                          {new Date(asset.last_updated).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </Table>
              </div>

              {filteredAssets.length === 0 && !loading && (
                <div className="text-center py-4">
                  <p className="text-white-50">No assets found matching the current filters.</p>
                </div>
              )}
            </>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="outline-secondary" onClick={handleClose}>
            Close
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default ExploreAttackSurfaceModal; 