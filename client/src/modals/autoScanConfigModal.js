import { Modal, Button, Form, Spinner } from 'react-bootstrap';
import { useState, useEffect } from 'react';

function AutoScanConfigModal({ show, handleClose, config }) {
  const tools = [
    { id: 'amass', name: 'Amass' },
    { id: 'sublist3r', name: 'Sublist3r' },
    { id: 'assetfinder', name: 'Assetfinder' },
    { id: 'gau', name: 'GAU' },
    { id: 'ctl', name: 'CTL' },
    { id: 'subfinder', name: 'Subfinder' },
    { id: 'consolidate_httpx_round1', name: 'Consolidate & Live Web Servers (Round 1)' },
    { id: 'shuffledns', name: 'ShuffleDNS' },
    { id: 'cewl', name: 'CeWL' },
    { id: 'consolidate_httpx_round2', name: 'Consolidate & Live Web Servers (Round 2)' },
    { id: 'gospider', name: 'GoSpider' },
    { id: 'subdomainizer', name: 'Subdomainizer' },
    { id: 'consolidate_httpx_round3', name: 'Consolidate & Live Web Servers (Round 3)' },
    { id: 'nuclei_screenshot', name: 'Nuclei Screenshot' },
    { id: 'metadata', name: 'Metadata' }
  ];

  const defaultConfig = {
    amass: true, sublist3r: true, assetfinder: true, gau: true, ctl: true, subfinder: true, consolidate_httpx_round1: true, shuffledns: true, cewl: true, consolidate_httpx_round2: true, gospider: true, subdomainizer: true, consolidate_httpx_round3: true, nuclei_screenshot: true, metadata: true, maxConsolidatedSubdomains: 2500, maxLiveWebServers: 500
  };

  const [localConfig, setLocalConfig] = useState(defaultConfig);
  const [loading, setLoading] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (config) {
      setLocalConfig({ ...defaultConfig, ...config });
    }
    setSaveSuccess(false);
    setError(null);
    setLoading(false);
  }, [config, show]);

  if (!localConfig) {
    return (
      <Modal show={show} onHide={handleClose} centered data-bs-theme="dark">
        <Modal.Body className="bg-dark text-center">
          <Spinner animation="border" variant="danger" />
        </Modal.Body>
      </Modal>
    );
  }

  const handleCheckboxChange = (toolId) => {
    setLocalConfig((prev) => ({ ...prev, [toolId]: !prev[toolId] }));
  };

  const handleSliderChange = (key, value) => {
    setLocalConfig((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    setLoading(true);
    setSaveSuccess(false);
    setError(null);
    try {
      const response = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-config`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(localConfig)
        }
      );
      if (!response.ok) {
        throw new Error('Failed to save configuration');
      }
      setSaveSuccess(true);
      setTimeout(() => {
        handleClose();
      }, 1500);
    } catch (err) {
      setError('Failed to save configuration. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal show={show} onHide={handleClose} centered data-bs-theme="dark">
      <Modal.Header closeButton className="border-secondary">
        <Modal.Title className="text-danger">Auto Scan Configuration</Modal.Title>
      </Modal.Header>
      <Form onSubmit={handleSubmit}>
        <Modal.Body className="bg-dark">
          {error && <div className="alert alert-danger">{error}</div>}
          {saveSuccess && <div className="alert alert-success">Configuration saved successfully!</div>}
          <Form.Group className="mb-3">
            {tools.map((tool) => (
              <Form.Check
                key={tool.id}
                type="checkbox"
                id={tool.id}
                name={tool.id}
                label={tool.name}
                checked={!!localConfig[tool.id]}
                onChange={() => handleCheckboxChange(tool.id)}
                className="mb-2 text-danger custom-checkbox"
              />
            ))}
          </Form.Group>
          <hr className="text-secondary" />
          <Form.Group className="mb-4">
            <div className="d-flex justify-content-between align-items-center">
              <Form.Label className="text-danger mb-0">Max Consolidated Subdomains</Form.Label>
              <span className="text-white">{localConfig.maxConsolidatedSubdomains}</span>
            </div>
            <Form.Range
              min={1000}
              max={10000}
              step={100}
              value={localConfig.maxConsolidatedSubdomains}
              onChange={e => handleSliderChange('maxConsolidatedSubdomains', Number(e.target.value))}
            />
          </Form.Group>
          <Form.Group className="mb-2">
            <div className="d-flex justify-content-between align-items-center">
              <Form.Label className="text-danger mb-0">Max Live Web Servers</Form.Label>
              <span className="text-white">{localConfig.maxLiveWebServers}</span>
            </div>
            <Form.Range
              min={100}
              max={2500}
              step={50}
              value={localConfig.maxLiveWebServers}
              onChange={e => handleSliderChange('maxLiveWebServers', Number(e.target.value))}
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer className="border-secondary">
          <Button variant="outline-secondary" onClick={handleClose} disabled={loading}>
            Cancel
          </Button>
          <Button variant="outline-danger" type="submit" disabled={loading || saveSuccess}>
            {loading ? 'Saving...' : saveSuccess ? 'Saved!' : 'Save Configuration'}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
}

export default AutoScanConfigModal; 