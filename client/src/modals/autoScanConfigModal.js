import { Modal, Button, Form } from 'react-bootstrap';

function AutoScanConfigModal({ show, handleClose, onSave }) {
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

  const handleSubmit = (event) => {
    event.preventDefault();
    const formData = new FormData(event.target);
    const config = {};
    tools.forEach(tool => {
      config[tool.id] = formData.get(tool.id) === 'on';
    });
    onSave(config);
    handleClose();
  };

  return (
    <Modal show={show} onHide={handleClose} centered data-bs-theme="dark">
      <Modal.Header closeButton className="border-secondary">
        <Modal.Title className="text-danger">Auto Scan Configuration</Modal.Title>
      </Modal.Header>
      <Form onSubmit={handleSubmit}>
        <Modal.Body className="bg-dark">
          <Form.Group className="mb-3">
            {tools.map((tool) => (
              <Form.Check
                key={tool.id}
                type="checkbox"
                id={tool.id}
                name={tool.id}
                label={tool.name}
                defaultChecked
                className="mb-2 text-danger custom-checkbox"
              />
            ))}
          </Form.Group>
        </Modal.Body>
        <Modal.Footer className="border-secondary">
          <Button variant="outline-secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button variant="outline-danger" type="submit">
            Save Configuration
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
}

export default AutoScanConfigModal; 