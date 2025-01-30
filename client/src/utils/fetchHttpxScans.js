const fetchHttpxScans = async (activeTarget, setHttpxScans, setMostRecentHttpxScan, setMostRecentHttpxScanStatus) => {
  try {
    const response = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${activeTarget.id}/scans/httpx`
    );
    if (!response.ok) throw new Error('Failed to fetch httpx scans');

    const data = await response.json();
    
    setHttpxScans(data || []);
    if (!Array.isArray(data) || data.length === 0) return null;

    const mostRecentScan = data.reduce((latest, scan) => {
      const scanDate = new Date(scan.created_at);
      return scanDate > new Date(latest.created_at) ? scan : latest;
    }, data[0]);

    const scanDetailsResponse = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/httpx/${mostRecentScan.scan_id}`
    );
    if (!scanDetailsResponse.ok) throw new Error('Failed to fetch httpx scan details');

    const scanDetails = await scanDetailsResponse.json();
    setMostRecentHttpxScan(scanDetails);
    setMostRecentHttpxScanStatus(scanDetails.status);

    return scanDetails;
  } catch (error) {
    console.error('Error fetching httpx scan details:', error);
  }
}

export default fetchHttpxScans; 