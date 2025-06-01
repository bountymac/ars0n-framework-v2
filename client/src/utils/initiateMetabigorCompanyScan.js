const initiateMetabigorCompanyScan = async (
  activeTarget,
  monitorScanStatus,
  setIsScanning,
  setScans,
  setMostRecentScanStatus,
  setMostRecentScan
) => {
  const companyName = activeTarget.scope_target;

  console.log(`Initiating Metabigor Company scan for: ${companyName}`);
  setIsScanning(true);

  try {
    const response = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/metabigor-company/run`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ company_name: companyName }),
      }
    );

    if (!response.ok) {
      throw new Error('Failed to initiate Metabigor Company scan');
    }

    const data = await response.json();
    console.log('Metabigor Company scan initiated:', data);

    monitorScanStatus(
      activeTarget,
      setScans,
      setMostRecentScan,
      setIsScanning,
      setMostRecentScanStatus
    );
  } catch (error) {
    console.error('Error initiating Metabigor Company scan:', error);
    setIsScanning(false);
  }
};

export default initiateMetabigorCompanyScan; 