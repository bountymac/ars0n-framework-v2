import { useState, useEffect } from 'react';
import { Modal, Button, Form, Row, Col } from 'react-bootstrap';

function APIKeysConfigModal({ show, handleClose, onOpenSettings, onApiKeySelected }) {
  const [apiKeys, setApiKeys] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState({
    SecurityTrails: '',
    Censys: '',
    Shodan: '',
    GitHub: ''
  });

  const tools = [
    { name: 'SecurityTrails', displayName: 'SecurityTrails' },
    { name: 'Censys', displayName: 'Censys CLI / API' },
    { name: 'Shodan', displayName: 'Shodan CLI / API' },
    { name: 'GitHub', displayName: 'GitHub Recon Tools' }
  ];

  useEffect(() => {
    if (show) {
      fetchApiKeys();
    }
  }, [show]);

  const fetchApiKeys = async () => {
    setLoading(true);
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/api-keys`
      );
      
      if (!response.ok) {
        throw new Error('Failed to fetch API keys');
      }
      
      const data = await response.json();
      setApiKeys(data || []);
    } catch (error) {
      console.error('Error fetching API keys:', error);
    } finally {
      setLoading(false);
    }
  };

  const getKeysForTool = (toolName) => {
    return apiKeys.filter(key => key.tool_name === toolName);
  };

  const handleKeySelect = (toolName, keyId) => {
    setSelectedKeys(prev => ({
      ...prev,
      [toolName]: keyId
    }));

    // Notify parent when SecurityTrails key is selected
    if (toolName === 'SecurityTrails' && keyId) {
      const selectedKey = apiKeys.find(key => key.id === keyId);
      onApiKeySelected?.(selectedKey?.key_values?.api_key ? true : false);
    }
  };

  const handleModalClose = () => {
    // Check if SecurityTrails key is selected before closing
    const selectedKey = apiKeys.find(key => key.id === selectedKeys.SecurityTrails);
    const hasSecurityTrailsKey = selectedKey?.key_values?.api_key ? true : false;
    onApiKeySelected?.(hasSecurityTrailsKey);
    handleClose();
  };

  const handleOpenSettingsModal = () => {
    handleModalClose();
    onOpenSettings();
  };

  return (
    <Modal data-bs-theme="dark" show={show} onHide={handleModalClose} size="lg">
      <Modal.Header closeButton>
        <Modal.Title className="text-danger">Configure API Keys</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <style>
          {`
            .custom-input {
              background-color: #343a40 !important;
              border: 1px solid #495057;
              color: #fff !important;
            }

            .custom-input:focus {
              border-color: #dc3545 !important;
              box-shadow: 0 0 0 0.2rem rgba(220, 53, 69, 0.25) !important;
            }

            .custom-input::placeholder {
              color: #6c757d !important;
            }
          `}
        </style>
        <p className="text-white-50 small mb-4">
          Select which API key to use for each tool. If no API key is available for a tool, you can add one using the Settings modal.
        </p>
        
        {loading ? (
          <div className="text-center py-4">
            <div className="spinner-border text-danger" role="status">
              <span className="visually-hidden">Loading...</span>
            </div>
          </div>
        ) : (
          <Row className="g-3">
            {tools.map((tool) => {
              const toolKeys = getKeysForTool(tool.name);
              
              return (
                <Col md={6} key={tool.name}>
                  <div className="border border-secondary rounded p-3">
                    <h6 className="text-danger mb-3">{tool.displayName}</h6>
                    
                    {toolKeys.length === 0 ? (
                      <div className="text-center">
                        <p className="text-white-50 small mb-3">No API keys available for this tool</p>
                        <Button 
                          variant="outline-danger" 
                          size="sm"
                          onClick={handleOpenSettingsModal}
                        >
                          Add API Key
                        </Button>
                      </div>
                    ) : (
                      <Form.Group>
                        <Form.Label className="text-white small">Select API Key:</Form.Label>
                        <Form.Select
                          value={selectedKeys[tool.name]}
                          onChange={(e) => handleKeySelect(tool.name, e.target.value)}
                          className="custom-input"
                        >
                          <option value="">-- Select Key --</option>
                          {toolKeys.map((key) => (
                            <option key={key.id} value={key.id}>
                              {key.api_key_name}
                            </option>
                          ))}
                        </Form.Select>
                      </Form.Group>
                    )}
                  </div>
                </Col>
              );
            })}
          </Row>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={handleModalClose}>
          Cancel
        </Button>
        <Button 
          variant="danger" 
          onClick={handleModalClose}
          disabled={loading}
        >
          Save Configuration
        </Button>
      </Modal.Footer>
    </Modal>
  );
}

export default APIKeysConfigModal; 