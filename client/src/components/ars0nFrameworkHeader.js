import { Row, Col, Button } from 'react-bootstrap';

function Ars0nFrameworkHeader() {
  return (
    <Row className="align-items-center mb-3">
      <Col xs="auto">
        <img src="/images/logo.avif" alt="Logo" style={{ height: '60px' }} />
      </Col>
      <Col xs="auto" className="ms-auto d-flex justify-content-end">
        <Button variant="link" className="text-white p-1">
          <i className="bi bi-question-circle" style={{ fontSize: '1.5rem' }}></i>
        </Button>
        <Button variant="link" className="text-white p-1">
          <i className="bi bi-person" style={{ fontSize: '1.5rem' }}></i>
        </Button>
        <Button variant="link" className="text-white p-1">
          <i className="bi bi-gear" style={{ fontSize: '1.5rem' }}></i>
        </Button>
      </Col>
    </Row>
  );
}

export default Ars0nFrameworkHeader;
