import React from 'react';
import { Modal, Table, Button } from 'react-bootstrap';
import { copyToClipboard } from '../utils/miscUtils';

export const MetabigorCompanyResultsModal = ({ 
  showMetabigorCompanyResultsModal, 
  handleCloseMetabigorCompanyResultsModal, 
  metabigorCompanyResults,
  setShowToast 
}) => {
  const formatResults = (results) => {
    if (!results?.result) return [];
    return results.result.split('\n').filter(line => line.trim());
  };

  const handleCopy = async () => {
    if (metabigorCompanyResults?.result) {
      const success = await copyToClipboard(metabigorCompanyResults.result);
      if (success && setShowToast) {
        setShowToast(true);
        setTimeout(() => setShowToast(false), 3000);
      }
    }
  };

  return (
    <Modal 
      show={showMetabigorCompanyResultsModal} 
      onHide={handleCloseMetabigorCompanyResultsModal} 
      size="lg"
      centered
    >
      <Modal.Header closeButton className="bg-dark text-white">
        <Modal.Title>Metabigor Company Scan Results</Modal.Title>
      </Modal.Header>
      <Modal.Body className="bg-dark text-white">
        {metabigorCompanyResults ? (
          <div>
            <div className="mb-3">
              <strong>Status:</strong> {metabigorCompanyResults.status}
            </div>
            <div className="mb-3">
              <strong>Company:</strong> {metabigorCompanyResults.company_name}
            </div>
            {metabigorCompanyResults.execution_time && (
              <div className="mb-3">
                <strong>Execution Time:</strong> {metabigorCompanyResults.execution_time}
              </div>
            )}
            {metabigorCompanyResults.result && (
              <div>
                <div className="d-flex justify-content-between align-items-center mb-3">
                  <strong>Root Domains Found ({formatResults(metabigorCompanyResults).length}):</strong>
                  <Button variant="outline-light" size="sm" onClick={handleCopy}>
                    Copy All
                  </Button>
                </div>
                <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
                  <Table striped bordered hover variant="dark" size="sm">
                    <thead>
                      <tr>
                        <th>#</th>
                        <th>Domain</th>
                      </tr>
                    </thead>
                    <tbody>
                      {formatResults(metabigorCompanyResults).map((domain, index) => (
                        <tr key={index}>
                          <td>{index + 1}</td>
                          <td className="font-monospace">{domain}</td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                </div>
              </div>
            )}
            {metabigorCompanyResults.error && (
              <div className="mt-3">
                <strong>Error:</strong>
                <pre className="bg-secondary p-2 mt-2 rounded">
                  {metabigorCompanyResults.error}
                </pre>
              </div>
            )}
          </div>
        ) : (
          <div>No results available</div>
        )}
      </Modal.Body>
      <Modal.Footer className="bg-dark">
        <Button variant="secondary" onClick={handleCloseMetabigorCompanyResultsModal}>
          Close
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export const MetabigorCompanyHistoryModal = ({ 
  showMetabigorCompanyHistoryModal, 
  handleCloseMetabigorCompanyHistoryModal, 
  metabigorCompanyScans 
}) => {
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const getStatusBadge = (status) => {
    const statusColors = {
      'completed': 'success',
      'failed': 'danger',
      'pending': 'warning',
      'running': 'info'
    };
    return statusColors[status] || 'secondary';
  };

  return (
    <Modal 
      show={showMetabigorCompanyHistoryModal} 
      onHide={handleCloseMetabigorCompanyHistoryModal} 
      size="xl"
      centered
    >
      <Modal.Header closeButton className="bg-dark text-white">
        <Modal.Title>Metabigor Company Scan History</Modal.Title>
      </Modal.Header>
      <Modal.Body className="bg-dark text-white">
        {metabigorCompanyScans && metabigorCompanyScans.length > 0 ? (
          <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
            <Table striped bordered hover variant="dark">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Company</th>
                  <th>Status</th>
                  <th>Domains Found</th>
                  <th>Execution Time</th>
                </tr>
              </thead>
              <tbody>
                {metabigorCompanyScans.map((scan, index) => (
                  <tr key={index}>
                    <td>{formatDate(scan.created_at)}</td>
                    <td>{scan.company_name}</td>
                    <td>
                      <span className={`badge bg-${getStatusBadge(scan.status)}`}>
                        {scan.status}
                      </span>
                    </td>
                    <td>
                      {scan.result ? scan.result.split('\n').filter(line => line.trim()).length : 0}
                    </td>
                    <td>{scan.execution_time || 'N/A'}</td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
        ) : (
          <div>No scan history available</div>
        )}
      </Modal.Body>
      <Modal.Footer className="bg-dark">
        <Button variant="secondary" onClick={handleCloseMetabigorCompanyHistoryModal}>
          Close
        </Button>
      </Modal.Footer>
    </Modal>
  );
}; 